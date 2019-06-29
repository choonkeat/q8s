package api

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type server struct {
	ctx         context.Context
	filename    string
	writeFile   *os.File
	messageSize int
	offset      int64
	mu          sync.Mutex
}

// RunPublisherConsumerServer runs a server, blocks until shutdown
func RunPublisherConsumerServer(addr, filename string, messageSize int) func(context.Context) error {
	return func(ctx context.Context) error {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		opts := []grpc.ServerOption{
			grpc.KeepaliveParams(keepalive.ServerParameters{
				Timeout: 3 * time.Second,
			}),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             3 * time.Second,
				PermitWithoutStream: false,
			}),
			grpc.MaxConcurrentStreams(1),
		}
		grpcServer := grpc.NewServer(opts...)

		writeFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			return err
		}
		defer writeFile.Close()
		defer writeFile.Sync()

		stat, err := writeFile.Stat()
		if err != nil {
			return err
		}
		// manually achieve "append" since `O_APPEND` means
		// behavior of Seek(negative) is undefined
		writeFile.Seek(stat.Size(), 0)

		server := server{
			ctx:         ctx,
			filename:    filename,
			writeFile:   writeFile,
			offset:      stat.Size(),
			messageSize: messageSize,
		}
		RegisterPublisherServer(grpcServer, &server)
		RegisterConsumerServer(grpcServer, &server)

		log.Printf("Listening %s", ln.Addr())
		defer log.Printf("Shutting down %s", ln.Addr())
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop()
		}()
		return grpcServer.Serve(ln)
	}
}

// Publish appends to s.writeFile
func (s *server) Publish(ctx context.Context, m *Message) (*Acknowledgement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	json.NewEncoder(os.Stdout).Encode(m)
	wrotelen, err := s.writeFile.Write(m.Data)
	if err != nil {
		s.writeFile.Seek(-int64(wrotelen), 2)
		return nil, logErr(errors.Wrapf(err, "only wrote %d bytes", wrotelen))
	}
	b := make([]byte, s.messageSize-wrotelen)
	padlen, err := s.writeFile.Write(b)
	if err != nil {
		s.writeFile.Seek(-int64(wrotelen+padlen), 2)
		return nil, logErr(errors.Wrapf(err, "only padded %d bytes", padlen))
	}
	s.offset = s.offset + int64(s.messageSize)
	return &Acknowledgement{
		Offset: s.offset,
	}, nil
}

// Consume reads and returns every `s.messageSize` bytes from `s.filename`, starting from `r.Offset`
func (s *server) Consume(r *ReadRequest, stream Consumer_ConsumeServer) error {
	f, err := os.OpenFile(s.filename, os.O_RDONLY, 06600)
	if err != nil {
		return logErr(errors.Wrapf(err, "open file %q", s.filename))
	}
	defer f.Close()
	offset := r.GetOffset()
	f.Seek(offset, 0)
	b := make([]byte, s.messageSize)
	for {
		readlen, err := f.Read(b)
		if err == io.EOF {
			select {
			case <-s.ctx.Done():
				return logErr(errors.Wrapf(s.ctx.Err(), "server"))
			case <-stream.Context().Done():
				return logErr(errors.Wrapf(stream.Context().Err(), "client"))
			case <-time.After(time.Second):
				// wait for file to append
				continue
			}
		}
		if err != nil {
			return logErr(errors.Wrapf(err, "f.Read"))
		}
		if readlen < s.messageSize {
			return logErr(errors.Wrapf(err, "only read %d bytes, expected %d bytes", readlen, s.messageSize))
		}
		if err := stream.Send(&ReadResponse{
			Data:       b,
			Offset:     offset,
			NextOffset: offset + int64(s.messageSize),
		}); err != nil {
			return logErr(errors.Wrapf(err, "stream send"))
		}
		offset = offset + int64(readlen)
	}
}

// see what error we return to clients
func logErr(err error) error {
	log.Println(err.Error())
	return err
}
