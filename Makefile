MSG=42
publisher: gomod protoc
	go run cmd/publisher/publisher.go -msg "$(MSG)"

OFFSET=0
consumer: gomod protoc
	go run cmd/consumer/consumer.go -offset $(OFFSET)

server: gomod protoc
	go run cmd/server/server.go

protoc:
	protoc api/message.proto --go_out=plugins=grpc:.

gomod:
	GO111MODULE=on go mod tidy -v
	GO111MODULE=on go mod vendor -v
