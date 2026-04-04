package app

import "context"

type Publisher interface {
	Publish(ctx context.Context, msg string) error
}
