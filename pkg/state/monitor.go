package state

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"time"
)

// Represents time
type Clock interface {
	Now() time.Time
}

type utcClock struct{}

func (utcClock) Now() time.Time { return time.Now().UTC() }

func NewUTCClock() *utcClock {
	return &utcClock{}
}

// AddMonitorStage adds a stage to the run group that updates the provided state reference when updates are received via the supplied channel.
func AddMonitorStage(g *run.Group, logger log.Logger, updates <-chan []*ec2.VpnConnection, clock Clock, state *State) {

	actorLogger := log.With(logger, "actor", "monitor state")

	stateMonitor := monitorActor(actorLogger, clock, state, updates)
	g.Add(stateMonitor.Execute, stateMonitor.Interrupt)

}

// monitorActor returns an actor that updates the provided state reference when updates are received via the supplied channel.
func monitorActor(logger log.Logger, clock Clock, updater Updater, updates <-chan []*ec2.VpnConnection) actor.Actor {

	cancel := make(chan struct{})

	return actor.NewActor(
		func() error {

			for {
				select {
				case vpnConnections := <-updates:
					_ = level.Debug(logger).Log("msg", "Got update")
					updater.Update(vpnConnections, clock.Now())

				case <-cancel:
					_ = level.Info(logger).Log("cancelled", "Asked to terminate")
					return nil
				}
			}
		},
		func(err error) {
			_ = level.Info(logger).Log("interrupted", fmt.Sprintf("interrupted with %v", err))
			close(cancel)
		},
	)

}
