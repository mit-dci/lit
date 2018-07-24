package nat

import (
	"log"
	"fmt"
	"github.com/jackpal/gateway"
	natpmp "github.com/jackpal/go-nat-pmp"
	"net"
	"time"
	"errors"
)

var (
	// private24BitBlock contains the set of private IPv4 addresses within
	// the 10.0.0.0/8 adddress space.
	private24BitBlock *net.IPNet

	// private20BitBlock contains the set of private IPv4 addresses within
	// the 172.16.0.0/12 address space.
	private20BitBlock *net.IPNet

	// private16BitBlock contains the set of private IPv4 addresses within
	// the 192.168.0.0/16 address space.
	private16BitBlock *net.IPNet

	// ErrMultipleNAT is an error returned when multiple NATs have been
	// detected.
	ErrMultipleNAT = errors.New("multiple NATs detected")
)

func init() {
	_, private24BitBlock, _ = net.ParseCIDR("10.0.0.0/8")
	_, private20BitBlock, _ = net.ParseCIDR("172.16.0.0/12")
	_, private16BitBlock, _ = net.ParseCIDR("192.168.0.0/16")
}

// ExternalIP returns the external IP address of the NAT-PMP enabled device.
// cant define this as a method of natpmp without definign a seaprate s truct
func ExternalIP(p *natpmp.Client) (net.IP, error) {
	res, err := p.GetExternalAddress()
	if err != nil {
		return nil, err
	}

	ip := net.IP(res.ExternalIPAddress[:])
	if isPrivateIP(ip) {
		return nil, fmt.Errorf("multiple NATs detected")
	}

	return ip, nil
}

// isPrivateIP determines if the IP is private.
func isPrivateIP(ip net.IP) bool {
	return private24BitBlock.Contains(ip) ||
		private20BitBlock.Contains(ip) || private16BitBlock.Contains(ip)
}

// within the given timeout.
func SetupPmp(timeout time.Duration, port uint16) (*natpmp.Client, error) {
	var err error
	// Retrieve the gateway IP address of the local network.
	gatewayIP, err := gateway.DiscoverGateway()
	if err != nil {
		return nil, err
	}

	pmp := natpmp.NewClientWithTimeout(gatewayIP, timeout)

	// We'll then attempt to retrieve the external IP address of this
	// device to ensure it is not behind multiple NATs.

	ip, err := ExternalIP(pmp)
	if err != nil {
		return nil, err
	}
	log.Printf("Your external IP is %s", ip)
	_, err = pmp.AddPortMapping("tcp", int(port), int(port), 0)
	if err != nil {
		return nil, err
	}

	return pmp, nil
}
