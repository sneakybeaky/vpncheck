package runtime

import (
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestShutdownActor(t *testing.T) {
	c := make(chan os.Signal, 1)

	underTest := shutdownActor(log.NewNopLogger(), c)

	// Execute the actor.
	errors := make(chan error)
	go func(a actor.Actor) {
		errors <- a.Execute()
	}(underTest)

	// Send a signal to shut down the actor
	c <- syscall.SIGINT

	select {
	case err := <-errors:

		if err == nil {
			t.Error("Shouldn't have been a nil error on shutdown")
			return
		}

		return
	case <-time.After(1 * time.Second):
	}

	t.Error("actor didn't shut down in response to interrupt")

}

var interruptests = []struct {
	name  string
	actor actor.Actor
}{
	{name: "Shutdown", actor: shutdownActor(log.NewNopLogger(), nil)},
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
