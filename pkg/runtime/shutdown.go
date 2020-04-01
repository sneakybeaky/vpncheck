package runtime

import (
	"fmt"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"os"
	"os/signal"
	"syscall"
)

// Shutdown just sits and waits for CTRL-C or shutdown signals
func Shutdown(g *run.Group, logger log.Logger) {

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	shutdown := shutdownActor(logger, c)
	g.Add(shutdown.Execute, shutdown.Interrupt)

}

func shutdownActor(logger log.Logger, c <-chan os.Signal) actor.Actor {

	cancel := make(chan struct{})

	return actor.NewActor(
		func() error {
			select {
			case sig := <-c:
				_ = level.Info(logger).Log("actor", "shutdown", "msg", fmt.Sprintf("received signal %s", sig))
				return fmt.Errorf("received signal %s", sig)
			case <-cancel:
				return nil
			}
		},
		func(err error) {
			_ = level.Info(logger).Log("actor", "shutdown", "msg", fmt.Sprintf("interrupted with: %v", err))
			close(cancel)
		},
	)

}
