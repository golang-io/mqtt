package main

import (
	"context"
	"fmt"
	"github.com/golang-io/mqtt"
	"github.com/golang-io/mqtt/packet"
	"golang.org/x/sync/errgroup"
	"log"
	"time"
)

func main() {
	group, ctx := errgroup.WithContext(context.Background())
	for i := 0; i < 100; i++ {
		i, c := i, mqtt.New(mqtt.URL("mqtt://127.0.0.1:1883"))

		group.Go(func() error {
			if err := c.Connect(ctx, fmt.Sprintf("%d", i)); err != nil {
				panic(err)
			}
			group.Go(func() error {

				timer := time.NewTimer(1 * time.Second)
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-timer.C:
						c.Publish(ctx, packet.Message{TopicName: fmt.Sprintf("topic-%d", i), Content: []byte("hello world")})
						timer.Reset(1 * time.Second)
					}
				}
			})
			return c.Subscribe(ctx, []packet.Subscription{
				{TopicFilter: "+"}, {TopicFilter: "a/b/c"},
			}, func(message packet.Message) error {
				log.Printf("id=%s, msg=%s", c.ID(), message)
				return nil
			})
		})

	}
	if err := group.Wait(); err != nil {
		panic(err)
	}
}
