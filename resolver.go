package vpcinfo

import (
	"context"
	"strings"
)

// Resolver is the interface implemented by *net.Resolver, or other resolvers
// used to lookup VPC information.
type Resolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

const (
	// DefaultDomain is the default domain used when initializing the default
	// resolver.
	DefaultDomain = "vpcinfo.local"
)

// ResolverWithDomain wraps the resolver passed as argument to append the given
// domain to all name lookups.
func ResolverWithDomain(domain string, resolver Resolver) Resolver {
	if domain == "" {
		return resolver
	}
	if !strings.HasPrefix(domain, ".") { // normalize
		domain = "." + domain
	}
	return &resolverWithDomain{
		resolver: resolver,
		domain:   domain,
	}
}

type resolverWithDomain struct {
	resolver Resolver
	domain   string
}

func (r *resolverWithDomain) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.resolver.LookupTXT(ctx, strings.TrimSuffix(name, ".")+r.domain)
}
