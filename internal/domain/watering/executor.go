package watering

import "context"

type Executor interface {
	Execute(ctx context.Context, w *Watering) error
}
