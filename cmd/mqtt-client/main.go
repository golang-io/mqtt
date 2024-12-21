package main

import (
	"context"
	"github.com/golang-io/mqtt"
	"github.com/golang-io/mqtt/packet"
	"log"
	"time"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := mqtt.New(mqtt.URL("mqtt://127.0.0.1:1883"), mqtt.Subscription(
		packet.Subscription{TopicFilter: "+"}, packet.Subscription{TopicFilter: "a/b/c"},
	))
	c.OnMessage(func(msg *packet.Message) {
		log.Printf(msg.String())
	})

	go func() {
		for {
			if err := c.SubmitMessage(&packet.Message{
				TopicName: "12345",
				Content:   []byte(time.Now().Format("2006-01-02 15:04:05")),
			}); err != nil {
				log.Printf("%v", err)
			}
			time.Sleep(time.Second)
		}

	}()

	if err := c.ConnectAndSubscribe(ctx); err != nil {
		log.Printf("%v", err)
		return
	}
}
