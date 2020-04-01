package actor

// Represents an actor as per https://github.com/oklog/run
// Actors are defined as a pair of functions: an execute function, which should run synchronously; and an interrupt function, which, when invoked, should cause the execute function to return
type actor struct {
	execute   func() error
	interrupt func(error)
}

func NewActor(execute func() error, interrupt func(error)) Actor {
	return actor{execute: execute, interrupt: interrupt}
}

func (a actor) Execute() error {
	return a.execute()
}

func (a actor) Interrupt(err error) {
	a.interrupt(err)
}

type Actor interface {
	Execute() error
	Interrupt(error)
}
