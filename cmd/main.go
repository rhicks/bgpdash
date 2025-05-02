package main

import (
	"bgp_dashboard/pkg" // Import the local BGP package
	"log"               // For logging errors and information
)

func main() {
	// Define Variables
	localRouterId := "192.0.2.1" // - Router ID: 192.0.2.1 (should be unique in the network, typically an IP address)
	localASN := 65001            // - ASN: 65001 (Autonomous System Number in private ASN range)
	remotePeerIP := "192.0.2.2"  // - Neighbor IP: 192.0.2.2 (the IP address of the remote BGP peer)
	remoteASN := 65002           // - Neighbor ASN: 65002 (the ASN of the remote BGP peer)

	// Create a new instance of the BGP service
	bgpService := pkg.NewBGPService()

	// Start the BGP server:
	err := bgpService.Start(localRouterId, uint32(localASN))
	if err != nil {
		log.Fatalf("Failed to start BGP server: %v", err)
	}

	// Configure a BGP peer/neighbor:
	err = bgpService.AddNeighbor(remotePeerIP, uint32(remoteASN))
	if err != nil {
		// If neighbor configuration fails, log the error and exit the program
		log.Fatalf("Failed to add neighbor: %v", err)
	}

	// Start monitoring BGP prefix updates in a separate goroutine
	// This allows the monitoring to run concurrently without blocking the main thread
	go bgpService.MonitorPrefixes()

	// Keep the program running indefinitely
	// This empty select statement blocks forever, preventing the program from exiting
	select {}
}
