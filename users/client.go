package users

import (
	"fmt"

	"github.com/weaveworks/flux"
)

// Client is a grpc client to the users service.
type Client struct{}

func NewClient(addr string) *Client {
	return &Client{}
}

// ConvertInternalInstanceIDToExternal implements instance.IDMapper
func (c *Client) ConvertInternalInstanceIDToExternal(inst flux.InstanceID) (string, error) {
	return "", fmt.Errorf("TODO: implement users.Client.ConvertInternalInstanceIDToExternal")
}

func (c *Client) Close() error {
	return nil
}
