package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/golang-io/mqtt"
	"golang.org/x/sync/errgroup"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	c := flag.String("config", "./config/dev.json", "Path to config file")

	flag.Parse()
	b, err := os.ReadFile(*c)
	if err != nil {
		log.Fatal(err)
	}
	if err = json.Unmarshal(b, &mqtt.CONFIG); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	group, ctx := errgroup.WithContext(context.Background())
	s := mqtt.NewServer(ctx)

	group.Go(func() error {
		if mqtt.CONFIG.MQTT.URL == "" {
			return nil
		}
		return s.ListenAndServe(mqtt.URL(mqtt.CONFIG.MQTT.URL))
	})

	// ca文件: ca.pem, 客户端证书: mqtt.pem, 客户端key文件: mqtt.key
	group.Go(func() error {
		if mqtt.CONFIG.MQTTs.URL == "" {
			return nil
		}
		return s.ListenAndServeTLS(mqtt.CONFIG.MQTTs.CertFile, mqtt.CONFIG.MQTTs.KeyFile, mqtt.URL(mqtt.CONFIG.MQTTs.URL))
	})
	group.Go(func() error {
		if mqtt.CONFIG.WebSocket.URL == "" {
			return nil
		}
		return s.ListenAndServeWebsocket(mqtt.URL(mqtt.CONFIG.WebSocket.URL))
	})
	group.Go(func() error {
		if mqtt.CONFIG.HTTP.URL == "" {
			return nil
		}
		return mqtt.Httpd()
	})
	err = group.Wait()
	log.Fatal(err)

}
