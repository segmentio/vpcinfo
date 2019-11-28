package vpcinfo

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Registry exposes VPC information to the program.
//
// Registry values are intended to be long-lived, they cache the results they
// observe from the resolver.
//
// Registry values are safe to use concurrently from multiple goroutines.
type Registry struct {
	// The resolver used by this registry.
	//
	// If nil, net.DefaultResolver is used instead.
	Resolver Resolver

	// Limits how long cached entries are retained before attempting to refresh
	// them from the resolver.
	//
	// Defaults to 1 minute.
	TTL time.Duration

	subnets cache // Subnets
}

// LookupSubnets returns the list of subnets in the VPC.
//
// The returned list of subnets is ordered by IP address and mask (smallest IP
// and largest mask first). Multiple calls to this method may return the same
// Subnets value, programs should treat it as a read-only value and avoid
// modifying it to prevent race conditions.
func (r *Registry) LookupSubnets(ctx context.Context) (Subnets, error) {
	v, err := r.subnets.load(time.Now(), r.ttl(), func() (interface{}, error) {
		records, err := r.resolver().LookupTXT(ctx, "subnets")
		if err != nil {
			return nil, err
		}

		subnets := make(Subnets, len(records))

		for i, r := range records {
			if err := parse(r, &subnets[i]); err != nil {
				return nil, err
			}
		}

		return subnets, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(Subnets), nil
}

func (r *Registry) resolver() Resolver {
	if r.Resolver != nil {
		return r.Resolver
	}
	return net.DefaultResolver
}

func (r *Registry) ttl() time.Duration {
	if r.TTL > 0 {
		return r.TTL
	}
	return time.Minute
}

type cache struct {
	value  atomic.Value // *value
	reload uint64
	mutex  sync.Mutex
}

type value struct {
	value interface{}
	err   error
	evict time.Time
}

func (c *cache) load(now time.Time, ttl time.Duration, lookup func() (interface{}, error)) (interface{}, error) {
	v, _ := c.value.Load().(*value)
	if v != nil {
		if now.Before(v.evict.Add(-ttl/2)) || !atomic.CompareAndSwapUint64(&c.reload, 0, 1) {
			return v.value, v.err
		}
		defer atomic.StoreUint64(&c.reload, 0)
	} else {
		// Block on the mutex to ensure that at most one goroutine sends a
		// request to lookup the value. Only when the cache had no value yet
		// would this code path be blocking, otherwise only one goroutine would
		// enter it due to the negotiation happening on incrementing the version.
		c.mutex.Lock()
		defer c.mutex.Unlock()

		if v, _ = c.value.Load().(*value); v != nil {
			return v.value, v.err
		}
	}

	val, err := lookup()
	// On error, retain the previous value if we are still within the TTL.
	if err != nil && v != nil && now.Before(v.evict) {
		val = v.value
		err = nil
	}

	v = &value{
		value: val,
		err:   err,
		evict: now.Add(ttl), // TODO: jitter?
	}
	c.value.Store(v)
	return v.value, v.err
}
