package metrics

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/oklog/pkg/group"
)

// UpdaterStage inserts calls to an Updater in a pipeline of VPN status update handlers
func UpdaterStage(logger log.Logger, updater Updater, in <-chan []*ec2.VpnConnection, out chan<- []*ec2.VpnConnection) actor.Actor {

	cancel := make(chan struct{})

	return actor.NewActor(func() error {

		for {

			select {

			case vpnStatus := <-in:
				updater.Update(vpnStatus)
				out <- vpnStatus

			case <-cancel:
				_ = level.Info(logger).Log("cancelled", "Asked to shut down")
				return nil
			}
		}
	},
		func(err error) {
			_ = level.Info(logger).Log("interrupted", "Received with: %v", err)
			close(cancel)
		})

}

// AddUpdaterStage adds an updater as a stage to the supplied run group
func AddUpdaterStage(group *group.Group, logger log.Logger, updater Updater, in <-chan []*ec2.VpnConnection, out chan<- []*ec2.VpnConnection) {

	actorLogger := log.With(logger, "actor", "vpn updater")

	u := UpdaterStage(actorLogger, updater, in, out)
	group.Add(u.Execute, u.Interrupt)

}
