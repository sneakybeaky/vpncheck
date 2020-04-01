package state

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
)

// Can update the status of a VPN connection
type Updater interface {
	Update(connections []*ec2.VpnConnection, timeStamp time.Time)
}

// State represents the last-known state of the VPN Connections.
type State struct {
	Connections []*ec2.VpnConnection
	Timestamp   time.Time
}

func (s *State) Update(connections []*ec2.VpnConnection, timeStamp time.Time) {
	s.Connections = connections
	s.Timestamp = timeStamp
}
