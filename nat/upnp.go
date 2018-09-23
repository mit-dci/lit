package nat

import (
	"context"

	"github.com/mit-dci/lit/logging"
	UpnP "github.com/NebulousLabs/go-UpnP"
)

func SetupUpnp(port uint16) error {
	// Connect to router
	deliver, err := UpnP.DiscoverCtx(context.Background())
	if err != nil {
		logging.Fatalf("Unable to discover router %v\n", err)
	}
	// Get external IP
	ip, err := deliver.ExternalIP()
	if err != nil {
		logging.Fatalf("Unable to get external ip %v\n", err)
	}
	logging.Infof("Your external IP is %s", ip)
	// Forward peer port
	err = deliver.Forward(uint16(port), "lnd peer port")
	if err != nil {
		logging.Fatalf("UpnP: Unable to forward pear port ip %v\n", err)
	}
	return nil
}
