package pkg

import (
	"context"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	"google.golang.org/protobuf/encoding/protojson"
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

// MonitorPrefixes establishes a real-time monitor for BGP route updates and displays them
// in both raw protobuf format and prettified JSON. The function runs continuously,
// processing BGP updates as they are received from peer routers.
func (s *BGPService) MonitorPrefixes() {
	// Initialize JSON marshaler with formatting options for better readability
	// Multiline ensures each field is on a new line
	// Indent specifies the spacing for nested structures
	marshaler := protojson.MarshalOptions{
		Multiline: true, // Format JSON across multiple lines
		Indent:    "  ", // Use two spaces for each level of indentation
	}

	// Begin watching for BGP events with specific filtering criteria
	err := s.server.WatchEvent(s.context, &api.WatchEventRequest{
		Table: &api.WatchEventRequest_Table{
			Filters: []*api.WatchEventRequest_Table_Filter{
				{
					// Filter for best paths only to avoid duplicate routes
					// This ensures we only see the winning BGP routes that are
					// actually being used for forwarding
					Type: api.WatchEventRequest_Table_Filter_BEST,
				},
			},
		},
	}, func(r *api.WatchEventResponse) {
		// Extract the routing table information from the response
		if table := r.GetTable(); table != nil {
			// Process each path (route) in the update
			for _, path := range table.Paths {
				// Display the raw protobuf message with all fields
				// The %+v format prints field names along with their values
				log.Printf("Received BGP Update:\n%+v\n", path)

				// Convert the protobuf message to formatted JSON
				// This provides a more structured and readable view of the update
				jsonBytes, err := marshaler.Marshal(path)
				if err != nil {
					log.Printf("Error marshaling to JSON: %v", err)
					continue
				}
				// Print the formatted JSON with proper indentation
				log.Printf("BGP Update in JSON format:\n%s\n", string(jsonBytes))
			}
		}
	})

	// Handle any errors that occur during the watch setup or execution
	// This includes connection issues or invalid message formats
	if err != nil {
		log.Printf("Error watching events: %v\n", err)
	}
}

// Stop gracefully shuts down the BGP server
func (s *BGPService) Stop() {
	s.server.Stop()
}
