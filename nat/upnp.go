package nat

import (
	"context"
	"fmt"

	"github.com/mit-dci/lit/logging"

	UpnP "github.com/NebulousLabs/go-UpnP"
)

func SetupUpnp(port uint16) error {
	// Connect to router
	deliver, err := UpnP.DiscoverCtx(context.Background())
	if err != nil {
		fmt.Printf("Unable to discover router %v\n", err)
		logging.Fatal(err)
	}
	// Get external IP
	ip, err := deliver.ExternalIP()
	if err != nil {
		fmt.Printf("Unable to get external ip %v\n", err)
		logging.Fatal(err)
	}
	logging.Infof("Your external IP is %s", ip)
	// Forward peer port
	err = deliver.Forward(uint16(port), "lnd peer port")
	if err != nil {
		fmt.Printf("UpnP: Unable to forward pear port ip %v\n", err)
		logging.Fatal(err)
	}
	return nil
}
