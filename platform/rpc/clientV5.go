package rpc

import (
	"io"

	"github.com/weaveworks/flux/platform"
)

// RPCClient is the rpc-backed implementation of a platform, for
// talking to remote daemons.
type RPCClientV5 struct {
	*RPCClientV4
}

var _ platform.PlatformV5 = &RPCClientV5{}

// NewClient creates a new rpc-backed implementation of the platform.
func NewClientV5(conn io.ReadWriteCloser) *RPCClientV5 {
	return &RPCClientV5{NewClientV4(conn)}
}

func (p *RPCClient) Sync(spec platform.SyncDef) error {
	var result SyncResult
	if err := p.client.Call("RPCServer.Sync", spec, &result); err != nil {
		if _, ok := err.(rpc.ServerError); !ok && err != nil {
			err = platform.FatalError{err}
		}
		return err
	}
	if len(result) > 0 {
		errs := platform.SyncError{}
		for id, msg := range result {
			errs[id] = errors.New(msg)
		}
		return errs
	}
	return nil
}
