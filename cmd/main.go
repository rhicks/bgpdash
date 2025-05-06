package main

import (
	// Import the local BGP package - this will be used to access the BGPService type
	"bgp_dashboard/pkg"
	// Import for logging - log package functions use pointers to output streams internally
	"log"
)

func main() {
	// Load configuration from YAML file
	config, err := pkg.LoadConfig("cmd/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a new instance of the BGP service
	// bgpService is likely a pointer (*BGPService) returned by NewBGPService()
	// We use a pointer here because:
	// 1. The service maintains state that needs to be modified
	// 2. We want to avoid copying the service structure
	// 3. Multiple methods need to work with the same instance
	bgpService := pkg.NewBGPService()

	// Start the BGP server
	// Using localRouterId as string (passed by value since strings are immutable)
	// uint32(localASN) is passed by value since it's a basic type
	// The method is called on bgpService pointer to modify the service state
	err = bgpService.Start(config.BGP.Local.RouterID, uint32(config.BGP.Local.ASN))
	if err != nil {
		// log.Fatalf internally handles pointer to the error interface
		// error interface is itself a pointer type in implementation
		log.Fatalf("Failed to start BGP server: %v", err)
	}

	// Configure a BGP peer/neighbor
	// remotePeerIP is passed by value (strings are immutable)
	// uint32(remoteASN) is passed by value (basic type)
	// Method called on bgpService pointer to modify internal state
	err = bgpService.AddNeighbor(config.BGP.Remote.PeerIP, uint32(config.BGP.Remote.ASN))
	if err != nil {
		// err is an interface (containing a pointer) passed to Fatalf
		log.Fatalf("Failed to add neighbor: %v", err)
	}

	// Start monitoring BGP prefix updates in a goroutine
	// Using a goroutine requires the bgpService pointer to be shared
	// This is safe because GoBGP handles concurrent access internally
	go bgpService.MonitorPrefixes()

	// Empty select{} blocks forever
	// No pointers/references needed as this is just a blocking statement
	// This prevents the program from exiting and garbage collecting our BGP service
	select {}
}
