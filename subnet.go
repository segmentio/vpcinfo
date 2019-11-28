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

// LookupAddr returns the subnet that addr belongs to.
func (subnets Subnets) LookupAddr(addr net.Addr) Subnet {
	var ip net.IP
	switch a := addr.(type) {
	case *net.TCPAddr:
		ip = a.IP
	case *net.UDPAddr:
		ip = a.IP
	case *net.IPAddr:
		ip = a.IP
	case *net.UnixAddr:
		ip = nil
	default:
		host, _, _ := net.SplitHostPort(a.String())
		ip = net.ParseIP(host)
	}
	return subnets.LookupIP(ip)
}

// LookupIP returns the subnet that ip belongs to
func (subnets Subnets) LookupIP(ip net.IP) Subnet {
	if ip != nil {
		for _, s := range subnets {
			if s.CIDR != nil && s.CIDR.Contains(ip) {
				return s
			}
		}
	}
	return Subnet{}
}
