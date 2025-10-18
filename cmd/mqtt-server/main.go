package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/golang-io/mqtt"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)

	c := flag.String("config", "./config/dev.json", "Path to config file")
	// addr := flag.String("mqtt", "mqtt://0.0.0.0:1883", "Address to listen on")
	flag.Parse()
	b, err := os.ReadFile(*c)
	if err != nil {
		log.Fatal(err)
	}
	if err = json.Unmarshal(b, &mqtt.CONFIG); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	s := mqtt.NewServer(context.Background())
	if err := s.InitServer(context.Background()); err != nil {
		log.Fatal(err)
	}
}
