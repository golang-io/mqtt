package main

import (
	"context"
	"github.com/golang-io/mqtt"
	"github.com/golang-io/mqtt/packet"
	"log"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := mqtt.New(mqtt.URL("mqtt://127.0.0.1:1883"))
	if err := c.Connect(ctx, "112233"); err != nil {
		panic(err)
	}

	if err := c.Subscribe(ctx, []packet.Subscription{
		{TopicFilter: "+"}, {TopicFilter: "a/b/c"},
	}, func(message packet.Message) error {
		//log.Printf("id=%s, msg=%s", message)
		return nil
	}); err != nil {
		log.Printf("%v", err)
		return
	}

}
