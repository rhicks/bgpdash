package pkg

import (
	// These imports provide types that are often reference types or contain pointers internally
	"context"                                       // context.Context is an interface type, often passed as-is
	api "github.com/osrg/gobgp/v3/api"              // api types are protobuf messages, used as pointers
	"github.com/osrg/gobgp/v3/pkg/server"           // server provides BGP functionality via pointer types
	"google.golang.org/protobuf/encoding/protojson" // used for JSON marshaling of pointer-based protobuf messages
	"log"                                           // provides logging functions that take interface{} arguments (often pointers)
)

// BGPService represents a BGP service instance with a server and context
// This struct is always used as a pointer (*BGPService) because:
// 1. It contains a pointer field (server)
// 2. Methods need to modify its state
// 3. It's shared between goroutines
type BGPService struct {
	server  *server.BgpServer // Pointer to server instance - required by GoBGP API
	context context.Context   // Interface type, internally may contain pointers
}

// NewBGPService creates and initializes a new BGP service
// Returns *BGPService (pointer) because:
// 1. Methods need to modify the service state
// 2. Multiple goroutines share this instance
// 3. Avoid copying the server pointer
func NewBGPService() *BGPService {
	return &BGPService{
		server:  server.NewBgpServer(), // Returns *BgpServer (pointer) as required by GoBGP
		context: context.Background(),  // Returns interface (may contain pointers internally)
	}
}

// Start initializes and starts the BGP server with the given router ID and ASN
// Uses pointer receiver (*BGPService) to modify server state
// Parameters are passed by value as they're small and immutable
func (s *BGPService) Start(routerId string, asn uint32) error {
	go s.server.Serve() // server pointer is safe to use across goroutines

	// StartBgp takes pointer to api.StartBgpRequest containing configuration
	// Global config is also a pointer as required by protobuf
	if err := s.server.StartBgp(s.context, &api.StartBgpRequest{
		Global: &api.Global{ // Pointer to protobuf message
			Asn:        asn,      // Value type (uint32)
			RouterId:   routerId, // Value type (string)
			ListenPort: 179,      // Value type (int)
		},
	}); err != nil {
		return err // error interface (contains pointer)
	}

	return nil
}

// AddNeighbor configures a new BGP peer with the specified address and ASN
// Uses pointer receiver to modify server state
// Parameters are passed by value (small, immutable types)
func (s *BGPService) AddNeighbor(neighborAddress string, neighborAsn uint32) error {
	// Create neighbor configuration
	// Uses pointers for protobuf messages as required by gRPC
	n := &api.Peer{
		Conf: &api.PeerConf{ // Nested pointer to protobuf message
			NeighborAddress: neighborAddress, // Value type (string)
			PeerAsn:         neighborAsn,     // Value type (uint32)
		},
	}

	// AddPeer takes pointer to request containing pointer to peer config
	return s.server.AddPeer(s.context, &api.AddPeerRequest{
		Peer: n, // Pointer to peer configuration
	})
}

// MonitorPrefixes establishes a real-time monitor for BGP route updates
// Uses pointer receiver to access server state
// Safe for concurrent use as server handles synchronization
func (s *BGPService) MonitorPrefixes() {
	// Value type as it's just configuration options
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}

	// WatchEvent takes pointer to request structure
	err := s.server.WatchEvent(s.context, &api.WatchEventRequest{
		Table: &api.WatchEventRequest_Table{ // Pointer to protobuf message
			Filters: []*api.WatchEventRequest_Table_Filter{ // Slice of pointers
				{
					Type: api.WatchEventRequest_Table_Filter_BEST,
				},
			},
		},
	}, func(r *api.WatchEventResponse) { // Callback receives pointer to response
		if table := r.GetTable(); table != nil {
			for _, path := range table.Paths { // Iterating over slice of pointers
				log.Printf("Received BGP Update:\n%+v\n", path)

				// Marshal converts pointer to protobuf to JSON
				jsonBytes, err := marshaler.Marshal(path)
				if err != nil {
					log.Printf("Error marshaling to JSON: %v", err)
					continue
				}
				log.Printf("BGP Update in JSON format:\n%s\n", string(jsonBytes))
			}
		}
	})

	if err != nil {
		log.Printf("Error watching events: %v\n", err) // err is interface containing pointer
	}
}

// Stop gracefully shuts down the BGP server
// Uses pointer receiver to modify server state
func (s *BGPService) Stop() {
	s.server.Stop() // Calls Stop on the server pointer
}
