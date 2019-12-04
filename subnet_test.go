package vpcinfo

import (
	"net"
	"testing"
)

var (
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
