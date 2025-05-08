package pkg

import "net"

// BGPUpdateMessage represents a comprehensive view of a BGP UPDATE message
type BGPUpdateMessage struct {
	// Withdrawn Routes
	WithdrawnRoutesLength uint16
	WithdrawnRoutes       []struct {
		PrefixLength uint8
		Prefix       net.IP
	}

	// Path Attributes
	TotalPathAttributeLength uint16

	Origin            *uint8 // 0=IGP, 1=EGP, 2=INCOMPLETE
	ASPath            [][]uint32
	NextHop           net.IP
	MED               *uint32
	LocalPref         *uint32
	AtomicAggregate   bool
	AggregatorAS      *uint32
	AggregatorAddress net.IP

	Communities         []uint32
	CommunityStrings    []string
	ExtendedCommunities [][]byte
	LargeCommunities    [][3]uint32

	// RPKI Origin Validation State (RFC 8097)
	RPKIValidationState *string

	// MP-BGP Extensions
	MPReachNLRI struct {
		AFI     uint16
		SAFI    uint8
		NextHop net.IP
		NLRIs   []struct {
			PrefixLength uint8
			Prefix       net.IP
		}
	}

	MPUnreachNLRI struct {
		AFI   uint16
		SAFI  uint8
		NLRIs []struct {
			PrefixLength uint8
			Prefix       net.IP
		}
	}

	// NLRI
	NLRI []struct {
		PrefixLength uint8
		Prefix       net.IP
	}

	// Metadata
	IsWithdraw bool
	FromPeer   string
	Timestamp  int64
}
