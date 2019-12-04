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

// Format writes subnets to w using the formatting verb v.
//
// The following formats are supported:
//
//     s    formats a human-readable summary of the subnets
//     v    formats the subnet list as a slice of strings
//
func (subnets Subnets) Format(w fmt.State, v rune) {
	switch v {
	case 's':
		zones := make(map[string]struct{})
		for _, s := range subnets {
			zones[s.Zone] = struct{}{}
		}
		fmt.Fprintf(w, "list of %d subnet(s) in %d zone(s)", len(subnets), len(zones))
	case 'v':
		fmt.Fprint(w, ([]Subnet)(subnets))
	}
}

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
