package soccer

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type publisher[T any] struct {
	// mx   sync.Mutex
	cMap sync.Map
}

func NewPublisher[T any]() *publisher[T] {
	return &publisher[T]{
		cMap: sync.Map{},
	}
}

// add consumer
func (pub *publisher[T]) AddConsumer(key int) *Consumer[T] {
	c := &Consumer[T]{
		eventC: make(chan T, 1),
	}
	v, loaded := pub.cMap.LoadOrStore(Key(key), c)
	if loaded {
		c = v.(*Consumer[T])
	}
	return c
}

// remove consumer
func (pub *publisher[T]) RemoveConsumer(key int) {
	v, loaded := pub.cMap.LoadAndDelete(Key(key))
	if loaded {
		c := v.(*Consumer[T])
		c.Close()
	}
}

// publish event into the subsriber channel
func (pub *publisher[T]) Publish(ctx context.Context, event T) error {
	eg := errgroup.Group{}
	pub.cMap.Range(func(key, value any) bool {
		eg.Go(func() error {
			c := value.(*Consumer[T])

			select {

			case c.eventC <- event:
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return ctx.Err()

			}
			return nil
		})

		return true
	})

	err := eg.Wait()
	return err
}

// close instance and all consumer
func (pub *publisher[T]) Close() error {
	pub.cMap.Range(func(key, value any) bool {
		c := value.(*Consumer[T])
		c.Close()
		return true
	})

	pub.cMap.Clear()
	return nil
}

// type key use to identify consumer
type Key int

// consumer only for retreieving event
type Consumer[T any] struct {
	eventC chan T
}

func (c *Consumer[T]) Close() error {
	close(c.eventC)
	return nil
}

// channel to get event
func (c *Consumer[T]) Event() <-chan T {
	return c.eventC
}
