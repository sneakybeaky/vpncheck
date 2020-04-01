package metrics

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/prometheus/client_golang/prometheus"
	"testing"
	"time"
)

var updatertests = []struct {
	name      string
	telemetry []*ec2.VpnConnection
}{
	{name: "Nil telemetry", telemetry: nil},
	{name: "Empty telemetry", telemetry: make([]*ec2.VpnConnection, 0)},
	{name: "One tunnel up", telemetry: testCaseFor(1, 0).telemetry},
}

func TestUpdateCalled(t *testing.T) {

	for _, tt := range updatertests {
		t.Run(tt.name, func(t *testing.T) {

			// Given undertest pipeline with one update sent to it
			updater := &capturingUpdater{}
			in := make(chan []*ec2.VpnConnection)
			out := make(chan []*ec2.VpnConnection)

			undertest := UpdaterStage(log.NewNopLogger(), updater, in, out)
			defer undertest.Interrupt(nil)
			go func() { in <- tt.telemetry }()

			// When the stage is running
			go func(a actor.Actor) {
				_ = a.Execute()
			}(undertest)

			var received []*ec2.VpnConnection
			select {
			case received = <-out:
			case <-time.After(1 * time.Second):
				t.Error("Timed out waiting for telemetry to be passed down the pipeline")
				return
			}

			// Then the update should be sent to the next stage in the pipeline
			if !cmp.Equal(tt.telemetry, received, cmpopts.IgnoreUnexported(ec2.VgwTelemetry{}, ec2.VpnConnection{})) {
				t.Errorf("Data sent to next stage incorrect : expected %v got %v", tt.telemetry, received)
				return
			}

			// And the update should have been sent to the Updater instance it wraps
			if sent := len(updater.captured); sent > 1 {
				t.Errorf("Too many invocation for the Updater : expected 1 got %d", sent)
				return
			}

			if !cmp.Equal(updater.captured[0], tt.telemetry, cmpopts.IgnoreUnexported(ec2.VgwTelemetry{}, ec2.VpnConnection{})) {
				t.Errorf("Data sent to the Updater incorrect : expected %v got %v", tt.telemetry, updater.captured[0])
				return
			}

		})
	}
}

type capturingUpdater struct {
	captured [][]*ec2.VpnConnection
}

func (c *capturingUpdater) Update(telemetry []*ec2.VpnConnection) {
	c.captured = append(c.captured, telemetry)
}

var vpnMetricActor = NewVpnStatusCollector(prometheus.NewRegistry(), log.NewNopLogger())

var interruptests = []struct {
	name  string
	actor actor.Actor
}{
	{name: "VPN metric publisher", actor: actor.NewActor(vpnMetricActor.Execute, vpnMetricActor.Interrupt)},
	{name: "Updater Stage", actor: updaterStageForTesting()},
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

func updaterStageForTesting() actor.Actor {

	// Given undertest pipeline with one update sent to it
	updater := &capturingUpdater{}
	in := make(chan []*ec2.VpnConnection)
	out := make(chan []*ec2.VpnConnection)

	return UpdaterStage(log.NewNopLogger(), updater, in, out)

}
