package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/choonkeat/q8s/api"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := errmain(); err != nil {
		log.Fatalln(err)
	}
}

func errmain() error {
	var addr string
	var filename string
	var messageSize int
	flag.StringVar(&addr, "addr", "localhost:8185", "address to listen")
	flag.StringVar(&filename, "filename", "file.log", "log flie")
	flag.IntVar(&messageSize, "message-size", 1024, "max size per message")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		cancel()
	}()

	return runCtxFuncs(ctx,
		api.RunPublisherConsumerServer(addr, filename, messageSize))
}

//
// Give me a list of `func(context.Context) error`. That. Is. All.
// Preferred.
//
func runCtxFuncs(parentCtx context.Context, services ...func(context.Context) error) error {
	g, ctx := errgroup.WithContext(parentCtx)

	for i := range services {
		service := services[i]
		g.Go(func() error {
			// if any service returns error, the shared `ctx` will be cancelled
			// which auto stops other services
			return service(ctx)
		})
	}

	// blocks until all [service func] have returned, then returns the first non-nil error (if any) from them.
	// https://godoc.org/golang.org/x/sync/errgroup#Group.Wait
	return g.Wait()
}
