package vpcinfo

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"time"
)

// DefaultRegistry is the default registry used by top-level functions of this
// package.
var DefaultRegistry = &Registry{
	Resolver: ResolverWithDomain(DefaultDomain, net.DefaultResolver),
	TTL:      time.Minute,
}

// LookupSubnets returns the list of subnets in the VPC.
func LookupSubnets() ([]Subnet, error) {
	return DefaultRegistry.LookupSubnets(context.Background())
}

func parse(s string, x interface{}) error {
	v := reflect.ValueOf(x).Elem()
	t := v.Type()

	fields := make(map[string][]int)

	for i, n := 0, t.NumField(); i < n; i++ {
		f := t.Field(i)
		s := f.Tag.Get("name")

		if s != "" {
			fields[s] = f.Index
		}
	}

	q, err := url.ParseQuery(s)
	if err != nil {
		return err
	}

	for name, values := range q {
		for _, value := range values {
			if f, ok := fields[name]; ok {
				field := v.FieldByIndex(f)
				switch x := field.Addr().Interface().(type) {
				case *string:
					*x = value
				case **net.IPNet:
					_, *x, err = net.ParseCIDR(value)
				default:
					// TODO: add types as needed
					err = fmt.Errorf("found unsupported field type: %s", field.Type())
				}
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
