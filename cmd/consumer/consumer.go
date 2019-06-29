package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/choonkeat/q8s/api"
	"google.golang.org/grpc"
)

func main() {
	if err := errmain(); err != nil {
		log.Fatalln(err)
	}
}

func errmain() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		cancel()
	}()

	var offset int64
	var addr string
	var maxConnectBackoffDelay time.Duration
	flag.Int64Var(&offset, "offset", 0, "offset to read from")
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
	client := api.NewConsumerClient(conn)

	stream, err := client.Consume(ctx, &api.ReadRequest{Offset: offset})
	if err != nil {
		log.Fatalln(err)
	}
	for {
		m, err := stream.Recv()
		if err != nil {
			log.Fatalln(err)
		}
		b := m.GetData()
		i := bytes.IndexByte(b, byte(0))
		if i > 0 {
			log.Printf("offset: %10d, data: %q", m.GetOffset(), string(b[:i]))
		}
	}
}
