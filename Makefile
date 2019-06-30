server: build/server
	./build/server

MSG=42
publisher: build/publisher
	./build/publisher -msg "$(MSG)"

OFFSET=0
consumer: build/consumer
	./build/consumer -offset $(OFFSET)

pprof-top:
	go tool pprof -top http://localhost:8184/debug/pprof/heap

#

build/server: go.sum api/message.pb.go
	go build -o build/server cmd/server/*.go

build/publisher: go.sum api/message.pb.go
	go build -o build/publisher cmd/publisher/*.go

build/consumer: go.sum api/message.pb.go
	go build -o build/consumer cmd/consumer/*.go

api/message.pb.go: api/message.proto
	protoc api/message.proto --go_out=plugins=grpc:.
	GO111MODULE=on go mod tidy -v
	GO111MODULE=on go mod vendor -v

go.sum: go.mod $(shell find . -iname '*.go')
	GO111MODULE=on go mod tidy -v
	GO111MODULE=on go mod vendor -v
	touch go.sum
