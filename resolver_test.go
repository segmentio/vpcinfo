package vpcinfo

import (
	"context"
	"reflect"
	"testing"
)

type resolverFunc func(context.Context, string) ([]string, error)

func (r resolverFunc) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r(ctx, name)
}

type resolverMap map[string][]string

func (r resolverMap) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r[name], ctx.Err()
}

func TestResolverWithDomain(t *testing.T) {
	r := ResolverWithDomain("somewhere.here", resolverMap{
		"name-1.somewhere.here": {"A", "B", "C"},
		"name-2.somewhere.here": {"1", "2", "3"},
		"name-3.somehwere.else": {"1", "2", "3"},
	})

	assertLookupTXT(t, r, "name-1", []string{"A", "B", "C"})
	assertLookupTXT(t, r, "name-2", []string{"1", "2", "3"})
	assertLookupTXT(t, r, "name-3", nil)
}

func assertLookupTXT(t *testing.T, r Resolver, name string, res []string) {
	t.Helper()

	if ret, err := r.LookupTXT(context.Background(), name); err != nil {
		t.Error("unexpected error retuurned by the resolver:", err)
	} else if !reflect.DeepEqual(ret, res) {
		t.Error("unexpected result returned by the resolver")
		t.Log("expected:", res)
		t.Log("found:   ", ret)
	}
}
