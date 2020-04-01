package state

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"time"

	"testing"
)

var fiveMinutes = 5 * time.Minute

var interruptests = []struct {
	name  string
	actor actor.Actor
}{
	{name: "State Monitor", actor: monitorActor(log.NewNopLogger(), NewUTCClock(), &State{}, make(chan []*ec2.VpnConnection))},
	{name: "AddPollerStage", actor: pollerActor(log.NewNopLogger(), make(chan []*ec2.VpnConnection, 1), newMockEC2Client(), &fiveMinutes)},
}

// Tests that the actors honour the contract as per https://github.com/oklog/run#run.
// When the interrupt function is called the actor should return
func TestInterrupt(t *testing.T) {

	for _, tt := range interruptests {
		t.Run(tt.name, func(t *testing.T) {

			underTest := tt.actor

			// Run the actor.
			errors := make(chan error)
			go func(a actor.Actor) {
				errors <- a.Execute()
			}(underTest)

			// Signal for the actor to stop
			underTest.Interrupt(nil)

			select {
			case <-errors:
				return
			case <-time.After(1 * time.Second):
			}

			t.Error("actor didn't shut down in response to interrupt")

		})
	}
}
