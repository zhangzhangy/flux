package rpc

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/weaveworks/flux/platform"
)

func pipes() (io.ReadWriteCloser, io.ReadWriteCloser) {
	type end struct {
		io.Reader
		io.WriteCloser
	}

	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()
	return end{clientReader, clientWriter}, end{serverReader, serverWriter}
}

func TestRPC(t *testing.T) {

	wrap := func(mock platform.Platform) platform.Platform {
		clientConn, serverConn := pipes()

		server, err := NewServer(mock)
		if err != nil {
			t.Fatal(err)
		}
		go server.ServeConn(serverConn)
		return NewClientV5(clientConn)
	}
	platform.PlatformTestBattery(t, wrap)
}

// ---

type poorReader struct{}

func (r poorReader) Read(p []byte) (int, error) {
	return 0, errors.New("failure to read")
}

// Return a pair of connections made of pipes, in which the first
// connection will fail Reads.
func faultyPipes() (io.ReadWriteCloser, io.ReadWriteCloser) {
	type end struct {
		io.Reader
		io.WriteCloser
	}

	serverReader, clientWriter := io.Pipe()
	_, serverWriter := io.Pipe()
	return end{poorReader{}, clientWriter}, end{serverReader, serverWriter}
}

func TestBadRPC(t *testing.T) {
	mock := &platform.MockPlatform{}
	clientConn, serverConn := faultyPipes()
	server, err := NewServer(mock)
	if err != nil {
		t.Fatal(err)
	}
	go server.ServeConn(serverConn)

	client := NewClientV5(clientConn)
	if err = client.Ping(); err == nil {
		t.Error("expected error from RPC system, got nil")
	}
	if _, ok := err.(platform.FatalError); !ok {
		t.Errorf("expected platform.FatalError from RPC mechanism, got %s", reflect.TypeOf(err))
	}
}
