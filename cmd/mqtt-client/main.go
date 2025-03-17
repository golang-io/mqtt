package main

import (
	"context"
	"fmt"
	"github.com/golang-io/mqtt"
	"github.com/golang-io/mqtt/packet"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := mqtt.New(mqtt.URL("mqtt://127.0.0.1:1883"), mqtt.Subscription(
		packet.Subscription{TopicFilter: "+"}, packet.Subscription{TopicFilter: "a/b/c"},
	))
	c.OnMessage(func(msg *packet.Message) {
		log.Printf("on: %s", msg.String())
	})
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if err := c.SubmitMessage(&packet.Message{
				TopicName: "12345",
				Content:   []byte(time.Now().Format("2006-01-02 15:04:05")),
			}); err != nil {
				log.Printf("%v", err)
			}
			time.Sleep(time.Second)
		}
	})

	group.Go(func() error {
		defer cancel()
		ignore := make(chan os.Signal, 1)
		sign := make(chan os.Signal, 1)

		signal.Notify(ignore, syscall.SIGHUP) // 终端挂起或者控制进程终止(hang up)
		signal.Notify(sign, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

		select {
		case <-ctx.Done():
			log.Printf("ctx done")
			return ctx.Err()
		case sig := <-sign:
			return fmt.Errorf("got sign: %s", sig)
		}
	})

	group.Go(func() error {
		return c.ConnectAndSubscribe(ctx)
	})
	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}
}
