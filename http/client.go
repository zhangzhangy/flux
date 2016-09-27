package http

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/weaveworks/fluxy"
)

type client struct {
	client   *http.Client
	router   *mux.Router
	endpoint string
}

func NewClient(c *http.Client, router *mux.Router, endpoint string) flux.Service {
	return &client{
		client:   c,
		router:   router,
		endpoint: endpoint,
	}
}

func (c *client) ListServices(t flux.Token, namespace string) ([]flux.ServiceStatus, error) {
	return invokeListServices(c.client, c.router, c.endpoint, t, namespace)
}

func (c *client) ListImages(t flux.Token, s flux.ServiceSpec) ([]flux.ImageStatus, error) {
	return invokeListImages(c.client, c.router, c.endpoint, t, s)
}

func (c *client) PostRelease(t flux.Token, s flux.ReleaseJobSpec) (flux.ReleaseID, error) {
	return invokePostRelease(c.client, c.router, c.endpoint, t, s)
}

func (c *client) GetRelease(t flux.Token, id flux.ReleaseID) (flux.ReleaseJob, error) {
	return invokeGetRelease(c.client, c.router, c.endpoint, t, id)
}

func (c *client) Automate(t flux.Token, id flux.ServiceID) error {
	return invokeAutomate(c.client, c.router, c.endpoint, t, id)
}

func (c *client) Deautomate(t flux.Token, id flux.ServiceID) error {
	return invokeDeautomate(c.client, c.router, c.endpoint, t, id)
}

func (c *client) History(t flux.Token, s flux.ServiceSpec) ([]flux.HistoryEntry, error) {
	return invokeHistory(c.client, c.router, c.endpoint, t, s)
}
