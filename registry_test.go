package vpcinfo

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	testEndpoints = [...]string{
		"subnets=subnets",
	}

	testTXT = [...]string{
		"subnet=subnet-075800de7fdd67afb&cidr=10.31.192.0/20&zone=us-west-2a",
		"subnet=subnet-08adfd56c276e8e66&cidr=10.31.224.0/20&zone=us-west-2c",
		"subnet=subnet-0c5e27d71f3d864df&cidr=10.31.128.0/19&zone=us-west-2b",
		"subnet=subnet-0cdc8bac1e690907e&cidr=10.31.160.0/19&zone=us-west-2c",
		"subnet=subnet-0e2cb67652c53230a&cidr=10.30.176.0/22&zone=us-west-2c",
		"subnet=subnet-0fde260c8d112c3b9&cidr=10.30.112.0/22&zone=us-west-2b",
		"subnet=subnet-752a9602&cidr=10.30.0.0/19&zone=us-west-2a",
		"subnet=subnet-762a9601&cidr=10.30.32.0/20&zone=us-west-2a",
		"subnet=subnet-a97de3cc&cidr=10.30.64.0/19&zone=us-west-2b",
		"subnet=subnet-ae7de3cb&cidr=10.30.96.0/20&zone=us-west-2b",
		"subnet=subnet-944497cd&cidr=10.30.160.0/20&zone=us-west-2c",
		"subnet=subnet-974497ce&cidr=10.30.128.0/19&zone=us-west-2c",
		"subnet=subnet-0eadfb8215baa9884&cidr=10.31.0.0/19&zone=us-west-2a",
		"subnet=subnet-02c1ac1322c424588&cidr=10.30.60.0/22&zone=us-west-2c",
		"subnet=subnet-03232a0956be0382c&cidr=10.31.96.0/19&zone=us-west-2a",
		"subnet=subnet-06e3c56c9df0b4615&cidr=10.30.52.0/22&zone=us-west-2a",
		"subnet=subnet-09042eb07f86172c4&cidr=10.31.32.0/19&zone=us-west-2b",
		"subnet=subnet-0aa52b33fb85b5e3a&cidr=10.30.56.0/22&zone=us-west-2b",
		"subnet=subnet-0c41f4c21182b5c5a&cidr=10.31.64.0/19&zone=us-west-2c",
		"subnet=subnet-0cc337a1ea0acfb65&cidr=10.30.48.0/22&zone=us-west-2a",
		"subnet=subnet-00958033a6eccb48f&cidr=10.31.208.0/20&zone=us-west-2b",
	}
)

func TestRegistryLookupPlatform(t *testing.T) {
	p, err := LookupPlatform()
	if err != nil {
		if os.IsPermission(err) {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	switch p.(type) {
	case aws, unknown:
		t.Log("platform:", p)
	default:
		t.Error("unrecognized platform:", p)
	}
}

func TestRegistryLookupZone(t *testing.T) {
	z, err := LookupZone()
	if err != nil {
		if os.IsPermission(err) {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Log("zone:", z)
}

func TestRegistryLookupSubnets(t *testing.T) {
	cacheMisses := uint32(0)

	r := &Registry{
		Resolver: resolverFunc(func(ctx context.Context, name string) ([]string, error) {
			switch name {
			case "":
				return testEndpoints[:], nil
			case "subnets":
				atomic.AddUint32(&cacheMisses, 1)
				delay := time.NewTimer(10 * time.Millisecond)
				defer delay.Stop()
				select {
				case <-delay.C:
					return testTXT[:], nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			default:
				return nil, fmt.Errorf("unknown vpc resource: %s", name)
			}
		}),
		Timeout: 1 * time.Second,
		TTL:     100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const N = 100
	wg := sync.WaitGroup{}

	for a := 0; a < 2; a++ {
		for i := 0; i < N; i++ {
			wg.Add(1)
			go func(ctx context.Context) {
				defer wg.Done()
				if _, err := r.LookupSubnets(ctx); err != nil {
					t.Error(err)
				}
			}(ctx)
		}

		wg.Wait()

		if a == 0 {
			time.Sleep(120 * time.Millisecond) // wait so the cache expires
		}
	}

	if miss := atomic.LoadUint32(&cacheMisses); miss != 2 {
		t.Error("invalid cache misses")
		t.Log("expected: 2")
		t.Log("found:   ", miss)
	}
}

func BenchmarkRegistry(b *testing.B) {
	r := &Registry{
		Resolver: resolverFunc(func(ctx context.Context, name string) ([]string, error) {
			switch name {
			case "":
				return testEndpoints[:], nil
			case "subnets":
				return testTXT[:], nil
			default:
				return nil, fmt.Errorf("unknown vpc resource: %s", name)
			}
		}),
	}

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, _ = r.LookupSubnets(ctx)
	}
}
