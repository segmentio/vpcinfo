package vpcinfo

import (
	"fmt"
	"net"
)

// Subnet values represent subnets.
type Subnet struct {
	ID   string     `name:"subnet"`
	Zone string     `name:"zone"`
	CIDR *net.IPNet `name:"cidr"`
}

func (s Subnet) String() string {
	return fmt.Sprintf("subnet=%s&cidr=%s&zone=%s", s.ID, s.CIDR, s.Zone)
}

// Subnets is a slice type which represents a list of subnets.
type Subnets []Subnet

// ZoneOf returns the zone that ip belongs to, or and empty string if it wasn't
// part of any subnets.
func (subnets Subnets) ZoneOf(ip net.IP) string {
	for _, s := range subnets {
		if s.CIDR != nil && s.CIDR.Contains(ip) {
			return s.Zone
		}
	}
	return ""
}
