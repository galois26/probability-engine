package probability

import "context"

// Engine is the public library interface consumed by applications such as
// multi-ingester.
type Engine interface {
	Assess(ctx context.Context, events []Event) ([]Assessment, error)
}

// EngineFunc adapts a function into an Engine.
type EngineFunc func(ctx context.Context, events []Event) ([]Assessment, error)

func (f EngineFunc) Assess(ctx context.Context, events []Event) ([]Assessment, error) {
	return f(ctx, events)
}
