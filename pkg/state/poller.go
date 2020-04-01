package state

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"time"
)

// AddPollerStage adds a stage to the run group that polls AWS for VPN telemetry data and sends down the status channel
func AddPollerStage(g *run.Group, logger log.Logger, status chan<- []*ec2.VpnConnection, svc ec2iface.EC2API, interval *time.Duration) {

	actorLogger := log.With(logger, "actor", "AWS poller")

	poller := pollerActor(actorLogger, status, svc, interval)
	g.Add(poller.Execute, poller.Interrupt)

}

// pollerActor polls AWS for VPN telemetry data and sends down the status channel
func pollerActor(logger log.Logger, status chan<- []*ec2.VpnConnection, svc ec2iface.EC2API, interval *time.Duration) actor.Actor {

	cancel := make(chan struct{})
	input := &ec2.DescribeVpnConnectionsInput{}
	ticker := time.NewTicker(*interval)

	return actor.NewActor(
		func() error {

			for {

				result, err := svc.DescribeVpnConnections(input)

				if err != nil {
					return err
				}

				status <- result.VpnConnections
				_ = level.Debug(logger).Log("msg", "Sent updated VPN telemetry data to next stage")

				select {

				case <-ticker.C:
					_ = level.Debug(logger).Log("msg", "Waking up")
					continue

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
