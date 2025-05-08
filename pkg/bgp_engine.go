package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	"log"
	"net"
)

const (
	RpkiValid    = 0
	RpkiNotFound = 1
	RpkiInvalid  = 2
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
		AfiSafis: []*api.AfiSafi{
			{
				Config: &api.AfiSafiConfig{
					Family: &api.Family{
						Afi:  api.Family_AFI_IP,
						Safi: api.Family_SAFI_UNICAST,
					},
					Enabled: true,
				},
				MpGracefulRestart: &api.MpGracefulRestart{
					Config: &api.MpGracefulRestartConfig{
						Enabled: true,
					},
				},
			},
		},
		Transport: &api.Transport{
			PassiveMode: false,
		},
		GracefulRestart: &api.GracefulRestart{
			Enabled:     true,
			RestartTime: 90,
			//LongLivedEnabled:    true,
			NotificationEnabled: true,
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
	err := s.server.WatchEvent(s.context, &api.WatchEventRequest{
		Table: &api.WatchEventRequest_Table{
			Filters: []*api.WatchEventRequest_Table_Filter{
				{
					Type: api.WatchEventRequest_Table_Filter_ADJIN,
				},
			},
		},
	}, func(r *api.WatchEventResponse) {
		if table := r.GetTable(); table != nil {
			for _, path := range table.Paths {
				var update BGPUpdateMessage
				update.FromPeer = path.GetNeighborIp()
				update.Timestamp = path.GetAge().GetSeconds()
				update.IsWithdraw = path.IsWithdraw

				// Zero/empty initializations
				update.NextHop = net.IP{}
				update.Origin = nil
				update.MED = nil
				update.LocalPref = nil
				update.AggregatorAS = nil
				update.AggregatorAddress = nil
				update.Communities = []uint32{}
				update.CommunityStrings = []string{}
				update.ExtendedCommunities = [][]byte{}
				update.LargeCommunities = [][3]uint32{}
				update.ASPath = [][]uint32{}
				update.WithdrawnRoutes = []struct {
					PrefixLength uint8
					Prefix       net.IP
				}{}
				update.NLRI = []struct {
					PrefixLength uint8
					Prefix       net.IP
				}{}
				update.MPReachNLRI = struct {
					AFI     uint16
					SAFI    uint8
					NextHop net.IP
					NLRIs   []struct {
						PrefixLength uint8
						Prefix       net.IP
					}
				}{}
				update.MPUnreachNLRI = struct {
					AFI   uint16
					SAFI  uint8
					NLRIs []struct {
						PrefixLength uint8
						Prefix       net.IP
					}
				}{}

				// Extract attributes
				for _, attr := range path.GetPattrs() {
					if nh := new(api.NextHopAttribute); attr.UnmarshalTo(nh) == nil {
						update.NextHop = net.ParseIP(nh.NextHop)
					}
					if origin := new(api.OriginAttribute); attr.UnmarshalTo(origin) == nil {
						u8 := uint8(origin.Origin)
						update.Origin = &u8
					}
					if med := new(api.MultiExitDiscAttribute); attr.UnmarshalTo(med) == nil {
						m := med.Med
						update.MED = &m
					}
					if lp := new(api.LocalPrefAttribute); attr.UnmarshalTo(lp) == nil {
						l := lp.LocalPref
						update.LocalPref = &l
					}
					if agg := new(api.AggregatorAttribute); attr.UnmarshalTo(agg) == nil {
						update.AggregatorAS = &agg.Asn
						update.AggregatorAddress = net.ParseIP(agg.Address)
					}
					if comm := new(api.CommunitiesAttribute); attr.UnmarshalTo(comm) == nil {
						update.Communities = comm.Communities
						for _, c := range comm.Communities {
							asn := c >> 16
							local := c & 0xFFFF
							update.CommunityStrings = append(update.CommunityStrings, fmt.Sprintf("%d:%d", asn, local))
						}
					}
					if extComm := new(api.ExtendedCommunitiesAttribute); attr.UnmarshalTo(extComm) == nil {
						for _, c := range extComm.Communities {
							if c != nil {
								update.ExtendedCommunities = append(update.ExtendedCommunities, c.Value)
							}
						}
					}
					if largeComm := new(api.LargeCommunitiesAttribute); attr.UnmarshalTo(largeComm) == nil {
						for _, c := range largeComm.Communities {
							update.LargeCommunities = append(update.LargeCommunities, [3]uint32{c.GlobalAdmin, c.LocalData1, c.LocalData2})
						}
					}
					// Handle AS_PATH attribute
					if asPath := new(api.AsPathAttribute); attr.UnmarshalTo(asPath) == nil {
						for _, segment := range asPath.Segments {
							update.ASPath = append(update.ASPath, segment.Numbers)
						}
					}
				}

				// Extract NLRI
				var nlri api.IPAddressPrefix
				if err := path.GetNlri().UnmarshalTo(&nlri); err == nil {
					update.NLRI = append(update.NLRI, struct {
						PrefixLength uint8
						Prefix       net.IP
					}{
						PrefixLength: uint8(nlri.PrefixLen),
						Prefix:       net.ParseIP(nlri.Prefix),
					})
				}

				// RPKI validation state
				switch path.GetValidation().GetState() {
				case RpkiValid:
					state := "valid"
					update.RPKIValidationState = &state
				case RpkiInvalid:
					state := "invalid"
					update.RPKIValidationState = &state
				case RpkiNotFound:
					state := "not-found"
					update.RPKIValidationState = &state
				}

				if jsonBytes, err := json.MarshalIndent(update, "", "  "); err == nil {
					log.Printf("BGP Update JSON:\n%s", string(jsonBytes))
				} else {
					log.Printf("Error marshalling update to JSON: %v", err)
				}
			}
		}
	})

	if err != nil {
		log.Printf("Error watching events: %v\n", err)
	}
}

// Stop gracefully shuts down the BGP server
// Uses pointer receiver to modify server state
func (s *BGPService) Stop() {
	s.server.Stop() // Calls Stop on the server pointer
}
