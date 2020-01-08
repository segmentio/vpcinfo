// Package vpcinfo provides APIs to extract VPC information from DNS resolvers.
//
// The library abstract parsing and caching of VPC information resolved from TXT
// records, as configured by the terraform modules in this repository.
package vpcinfo

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
)

// DefaultRegistry is the default registry used by top-level functions of this
// package.
var DefaultRegistry = &Registry{
	Resolver: ResolverWithDomain(DefaultDomain, net.DefaultResolver),
	Timeout:  2 * time.Second,
	TTL:      1 * time.Minute,
}

// LookupPlatform returns the name of the VPC platform.
func LookupPlatform() (Platform, error) {
	return DefaultRegistry.LookupPlatform(context.Background())
}

// LookupSubnets returns the list of subnets in the VPC.
func LookupSubnets() (Subnets, error) {
	return DefaultRegistry.LookupSubnets(context.Background())
}

// LookupZone returns the name of the current VPC zone.
func LookupZone() (Zone, error) {
	return DefaultRegistry.LookupZone(context.Background())
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

func splitNameAndValue(s string) (string, string) {
	i := strings.IndexByte(s, '=')
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

// Platform is an interface representing the VPC platform that the program is
// running.
type Platform interface {
	// Returns the name of the platform.
	String() string
	// Returns the zone that the program is running in.
	LookupZone(context.Context) (Zone, error)
}

// Zone is a string type representing infrastructure zones.
type Zone string

// String returns z as a string value, satisfies the fmt.Stringer interface.
func (z Zone) String() string { return string(z) }

func whereAmI() (Platform, error) {
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/identify_ec2_instances.html
	for _, path := range [...]string{
		"/sys/devices/virtual/dmi/id/product_uuid",
		"/sys/hypervisor/uuid",
	} {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		s := string(b)
		switch {
		case strings.HasPrefix(s, "EC2"), strings.HasPrefix(s, "ec2"):
			return aws{}, nil
		}
	}
	return unknown{}, nil
}

type aws struct{}

func (aws) String() string { return "aws" }

func (aws) LookupZone(ctx context.Context) (Zone, error) {
	r, err := httpClient.Get("http://169.254.169.254/latest/meta-data/placement/availability-zone")
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	return Zone(b), err
}

type unknown struct{}

func (unknown) String() string { return "unknown" }

func (unknown) LookupZone(ctx context.Context) (Zone, error) { return "", ctx.Err() }

var httpClient = &http.Client{
	Transport: &http.Transport{
		DisableCompression: true,
		DisableKeepAlives:  true,
	},
}

type endpoints map[string]string

func (e endpoints) String() string {
	return fmt.Sprintf("list of %d resource(s)", len(e))
}

type multiPlatform []Platform

func (p multiPlatform) String() string { return "multi" }

func (p multiPlatform) LookupZone(ctx context.Context) (Zone, error) {
	for _, platform := range p {
		if zone, err := platform.LookupZone(ctx); err == nil {
			return zone, nil
		}
	}

	return "", fmt.Errorf("no zone found.")
}

type networkInterface struct {
	Subnets []Subnet
}

func (p networkInterface) String() string { return "networkInterface" }

func (p networkInterface) LookupZone(ctx context.Context) (Zone, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		for _, subnet := range p.Subnets {
			if subnet.CIDR.Contains(net.ParseIP(addr.String())) {
				return Zone(subnet.Zone), nil
			}
		}
	}

	return "", fmt.Errorf("ip addresses don't match subnets.")
}
