package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/choonkeat/q8s/api"
	"google.golang.org/grpc"
)

func main() {
	var msgData string
	var addr string
	var maxConnectBackoffDelay time.Duration
	flag.StringVar(&msgData, "msg", "42", "message to send")
	flag.StringVar(&addr, "addr", "localhost:8185", "address to connect to")
	flag.DurationVar(&maxConnectBackoffDelay, "max-connect-backoff-delay", 5*time.Second, "maximum delay between retries to connect")
	flag.Parse()

	conn, err := grpc.Dial(addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithBackoffConfig(grpc.BackoffConfig{MaxDelay: maxConnectBackoffDelay}),
	)
	if err != nil {
		log.Fatalln(err)
	}
	client := api.NewPublisherClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ack, err := client.Publish(ctx, &api.Message{
		Data: []byte(msgData),
	})
	if err != nil {
		log.Fatalln(err)
	}
	json.NewEncoder(os.Stdout).Encode(ack)
}
