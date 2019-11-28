package vpcinfo

import (
	"net"
	"testing"
)

var (
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

	testSubnets Subnets
)

func init() {
	testSubnets = make(Subnets, len(testTXT))

	for i, s := range testTXT {
		if err := parse(s, &testSubnets[i]); err != nil {
			panic(err)
		}
	}
}

func TestSubnets(t *testing.T) {
	for _, s := range testSubnets {
		ip := make(net.IP, len(s.CIDR.IP))
		copy(ip, s.CIDR.IP)
		ip[len(ip)-1] = 1

		if x := testSubnets.LookupAddr(&net.TCPAddr{IP: ip, Port: 4242}); x != s {
			t.Error("subnet mismatch for", ip)
			t.Log("expected:", s)
			t.Log("found:   ", x)
		}
	}
}

func BenchmarkSubnetsZoneOf(b *testing.B) {
	ip := net.ParseIP("10.31.32.33")

	for i := 0; i < b.N; i++ {
		_ = testSubnets.LookupIP(ip)
	}
}
