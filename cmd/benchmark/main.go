package main

import (
	"context"
	"fmt"
	"github.com/golang-io/mqtt"
	"github.com/golang-io/mqtt/packet"
	"golang.org/x/sync/errgroup"
	"time"
)

func main() {
	group, ctx := errgroup.WithContext(context.Background())
	for i := 0; i < 1; i++ {
		time.Sleep(1 * time.Second)
		i, c := i, mqtt.New(
			mqtt.URL("mqtt://127.0.0.1:1883"),
			mqtt.Subscription(
				packet.Subscription{TopicFilter: "+"},
				packet.Subscription{TopicFilter: "a/b/c"},
			),
		)
		c.OnMessage(func(message *packet.Message) {
			//log.Printf("id=%s, msg=%s", c.ID(), message)
			fmt.Printf(".")
		})
		group.Go(func() error {
			timer := time.NewTimer(1 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-timer.C:
					for j := 0; j < 100; j++ {
						c.SubmitMessage(&packet.Message{
							TopicName: fmt.Sprintf("topic-%02d", i),
							Content:   []byte("hello world"),
						})
					}

					timer.Reset(1 * time.Second)
				}
			}
		})
		group.Go(func() error {
			return c.ConnectAndSubscribe(ctx)
		})
	}
	if err := group.Wait(); err != nil {
		panic(err)
	}
}
