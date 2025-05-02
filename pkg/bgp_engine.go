package pkg

import (
	"context"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	"log"
)

// BGPService represents a BGP service instance with a server and context
type BGPService struct {
	server  *server.BgpServer // The main BGP server instance
	context context.Context   // Context for managing BGP operations
}

// NewBGPService creates and initializes a new BGP service
func NewBGPService() *BGPService {
	return &BGPService{
		server:  server.NewBgpServer(), // Create a new BGP server instance
		context: context.Background(),  // Create a background context
	}
}

// Start initializes and starts the BGP server with the given router ID and ASN
func (s *BGPService) Start(routerId string, asn uint32) error {
	// Start the BGP server in a separate goroutine
	go s.server.Serve()

	// Configure and start the BGP process with global parameters
	if err := s.server.StartBgp(s.context, &api.StartBgpRequest{
		Global: &api.Global{
			Asn:        asn,      // Autonomous System Number
			RouterId:   routerId, // Router ID (typically an IP address)
			ListenPort: 179,      // Use default BGP port (179)
		},
	}); err != nil {
		return err
	}

	return nil
}

// AddNeighbor configures a new BGP peer with the specified address and ASN
func (s *BGPService) AddNeighbor(neighborAddress string, neighborAsn uint32) error {
	// Create neighbor configuration
	n := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: neighborAddress, // IP address of the BGP peer
			PeerAsn:         neighborAsn,     // ASN of the BGP peer
		},
	}

	// Add the peer to the BGP server
	return s.server.AddPeer(s.context, &api.AddPeerRequest{
		Peer: n,
	})
}

// MonitorPrefixes sets up a watch for BGP route updates
func (s *BGPService) MonitorPrefixes() {
	// Set up event watching with filters
	err := s.server.WatchEvent(s.context, &api.WatchEventRequest{
		Table: &api.WatchEventRequest_Table{
			Filters: []*api.WatchEventRequest_Table_Filter{
				{
					Type: api.WatchEventRequest_Table_Filter_BEST, // Only watch for best path updates
				},
			},
		},
	}, func(r *api.WatchEventResponse) {
		// Process each received event
		if table := r.GetTable(); table != nil {
			// Iterate through all paths in the update
			for _, path := range table.Paths {
				// Check if the path has NLRI (Network Layer Reachability Information)
				if nlri := path.GetNlri(); nlri != nil {
					// Log the received prefix
					log.Printf("Received prefix: %s\n", nlri.String())
				}
			}
		}
	})

	// Handle any errors during event watching
	if err != nil {
		log.Printf("Error watching events: %v\n", err)
	}
}

// Stop gracefully shuts down the BGP server
func (s *BGPService) Stop() {
	s.server.Stop()
}
