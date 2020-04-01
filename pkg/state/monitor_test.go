package state

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"testing"
	"time"
)

func TestStateUpdate(t *testing.T) {

	vpnState := &State{}

	// Given an update sent to a channel
	waiter := newStateWaiter(vpnState)
	updates := make(chan []*ec2.VpnConnection, 1)

	expectedClock := newFixedClock()
	underTest := monitorActor(log.NewNopLogger(), expectedClock, waiter, updates)
	defer underTest.Interrupt(nil)

	expectedGatewayId := aws.String("blahblahblah")
	expectedConnection := &ec2.VpnConnection{VpnGatewayId: expectedGatewayId}

	updates <- []*ec2.VpnConnection{expectedConnection}

	// When the actor is run
	go func(a actor.Actor) {
		_ = a.Execute()
	}(underTest)

	// Then the state should be modified to reflect the update
	select {
	case <-waiter.c:
		// expected - state has been updated
	case <-time.After(1 * time.Second):
		t.Errorf("State wasn't updated")
		return
	}

	if len(vpnState.Connections) != 1 {
		t.Errorf("Expected 1 update, but got %d", len(vpnState.Connections))
		return
	}

	if *vpnState.Connections[0].VpnGatewayId != *expectedGatewayId {
		t.Errorf("Updated connection state incorrect. Expected a gateway id of `%s` but got `%s`", *expectedGatewayId, *vpnState.Connections[0].VpnGatewayId)
	}

	if !vpnState.Timestamp.Equal(expectedClock.Now()) {
		t.Errorf("Updated timestamp of state incorrect. Expected a timestamp of `%s` but got `%s`", expectedClock.Now(), vpnState.Timestamp)
	}

}

type stateWaiter struct {
	decorated Updater
	c         chan struct{}
}

func (sw stateWaiter) Update(connections []*ec2.VpnConnection, timeStamp time.Time) {
	defer close(sw.c)
	sw.decorated.Update(connections, timeStamp)
}

func newStateWaiter(updater Updater) *stateWaiter {

	return &stateWaiter{
		decorated: updater,
		c:         make(chan struct{}),
	}
}

type fixedClock struct {
	fixedNow time.Time
}

func (f fixedClock) Now() time.Time { return f.fixedNow }

func newFixedClock() *fixedClock {
	return &fixedClock{
		fixedNow: time.Date(2009, 11, 17, 20, 34, 58, 0, time.UTC),
	}
}
