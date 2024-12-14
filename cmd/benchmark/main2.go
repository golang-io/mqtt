package main

import (
	"fmt"
	paho_mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang-io/requests"
	"log"
	"sync"
	"time"
)

var maxConn = 100

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	group := sync.WaitGroup{}
	for i := 0; i < maxConn; i++ {
		i := i
		group.Add(1)
		go func() {
			defer group.Done()
			pahoMqttStart(i)
		}()
	}
	group.Wait()

}

func onMessageReceived(client paho_mqtt.Client, message paho_mqtt.Message) {
	log.Printf("topic:%s, msg:%s", message.Topic(), message.Payload())
}

func pahoMqttStart(i int) {
	server := "tcp://127.0.0.1:1883"
	qos := byte(0x00)
	id := requests.GenId()
	connOpts := paho_mqtt.NewClientOptions().AddBroker(server).SetClientID(id).SetCleanSession(true)
	connOpts.SetAutoReconnect(false)

	client := paho_mqtt.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Printf("Connected to %s\n", server)

	if token := client.Subscribe("+", qos, onMessageReceived); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	timer := time.NewTimer(0 * time.Second)
	for {
		select {
		case <-timer.C:
			if t := client.Publish(fmt.Sprintf("topic_%02d", i), qos, false, fmt.Sprintf("paho_mqtt:test-%02d", i)); t.Wait() && t.Error() != nil {
				log.Println(t.Error()) // Use your preferred logging technique (or just fmt.Printf)
				panic(t.Error())
			}
			timer.Reset(time.Second)
		}
	}
}
