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

	// Time limit for blocking operations ran by this registry.
	//
	// Zero means to apply no limit.
	Timeout time.Duration

	// Limits how long cached entries are retained before attempting to refresh
	// them from the resolver.
	//
	// Defaults to 1 minute.
	TTL time.Duration

	subnets  cache // Subnets
	platform cache // Platform
	zone     cache // Zone
}

// LookupPlatform returns the name of the VPC platform, which will be either
// "aws" or "unknown".
func (r *Registry) LookupPlatform(ctx context.Context) (Platform, error) {
	v, err := r.platform.load(r.ttl(), func() (interface{}, error) {
		return whereAmI()
	})
	p, _ := v.(Platform)
	return p, err
}

// LookupSubnets returns the list of subnets in the VPC.
//
// Multiple calls to this method may return the same Subnets value, programs
// should treat it as a read-only value and avoid modifying it to prevent race
// conditions.
func (r *Registry) LookupSubnets(ctx context.Context) (Subnets, error) {
	v, err := r.subnets.load(r.ttl(), func() (interface{}, error) {
		ctx, cancel := r.withTimeout(ctx)
		defer cancel()

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
	s, _ := v.(Subnets)
	return s, err
}

// LookupZone returns the name of the VPC zone that the program is running in.
func (r *Registry) LookupZone(ctx context.Context) (Zone, error) {
	v, err := (r.zone.load(r.ttl(), func() (interface{}, error) {
		ctx, cancel := r.withTimeout(ctx)
		defer cancel()

		p, err := r.LookupPlatform(ctx)
		if err != nil {
			return nil, err
		}

		return p.LookupZone(ctx)
	}))
	z, _ := v.(Zone)
	return z, err
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

func (r *Registry) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.Timeout > 0 {
		return context.WithTimeout(ctx, r.Timeout)
	}
	return context.WithCancel(ctx)
}

type cache struct {
	value  atomic.Value // *value
	reload uint64
	mutex  sync.Mutex
}

type value struct {
	value  interface{}
	err    error
	update time.Time
	expire time.Time
}

func (c *cache) load(ttl time.Duration, lookup func() (interface{}, error)) (interface{}, error) {
	now := time.Now()

	v, _ := c.value.Load().(*value)
	if v != nil {
		if now.Before(v.update) || !atomic.CompareAndSwapUint64(&c.reload, 0, 1) {
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

	update := now.Add(ttl / 2)
	expire := now.Add(ttl)

	val, err := lookup()
	// On error, retain the previous value if we are still within the TTL.
	if err != nil && v != nil && now.Before(v.expire) {
		val = v.value
		err = nil
		// Keep the same expiration time so the cached value is eventually
		// removed if we keep failing to update it.
		expire = v.expire
	}

	v = &value{
		value:  val,
		err:    err,
		update: update,
		expire: expire,
	}

	c.value.Store(v)
	return v.value, v.err
}
