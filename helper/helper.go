package helper

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/pkg/errors"

	"github.com/weaveworks/fluxy"
	"github.com/weaveworks/fluxy/platform"
	"github.com/weaveworks/fluxy/platform/kubernetes"
	"github.com/weaveworks/fluxy/registry"
)

type Helper struct {
	platform *kubernetes.Cluster
	registry *registry.Client
	logger   log.Logger
	duration metrics.Histogram
}

func New(
	platform *kubernetes.Cluster,
	registry *registry.Client,
	logger log.Logger,
	duration metrics.Histogram,
) *Helper {
	return &Helper{
		platform: platform,
		registry: registry,
		logger:   logger,
		duration: duration,
	}
}

// Sometimes we care if we can't find the containers for a service,
// sometimes we just want the information we can get.
type ContainersOrExcuse struct {
	Excuse     error
	Containers []platform.Container
}

type Service struct {
	ID         flux.ServiceID
	Status     string
	Containers ContainersOrExcuse
}

func (s *Service) ContainersOrNil() []platform.Container {
	return s.Containers.Containers
}

func (s *Service) ContainersOrError() ([]platform.Container, error) {
	return s.Containers.Containers, s.Containers.Excuse
}

type ImageMap map[string][]flux.ImageDescription

// LatestImage returns the latest releasable image for a repository.
// A releasable image is one that is not tagged "latest". (Assumes the
// available images are in descending order of latestness.)
func (m ImageMap) LatestImage(repo string) (flux.ImageDescription, error) {
	for _, image := range m[repo] {
		_, _, tag := image.ID.Components()
		if strings.EqualFold(tag, "latest") {
			continue
		}
		return image, nil
	}
	return flux.ImageDescription{}, errors.New("no valid images available")
}

// %%TODO
// Get the services in `namespace` along with their containers (if
// there are any) from the platform; if namespace is blank, just get
// all the services, in any namespace.
func (h *Helper) GetAllServices(namespace string) ([]*Service, error) {
	return []*Service{}, nil
}

// %% TODO
// Get all services except those with an ID in the set given
func (h *Helper) GetAllServicesExcept(ignored flux.ServiceIDSet) ([]*Service, error) {
	return []*Service{}, nil
}

// %% TODO
// Get the services mentioned, along with their containers.
func (h *Helper) GetServices(ids []flux.ServiceID) ([]*Service, error) {
	return []*Service{}, nil
}

// %%% TODO
// Get the images available for the services given. An image may be
// mentioned more than once in the services, but will only be fetched
// once.
func (h *Helper) CollectAvailableImages(services []*Service) (ImageMap, error) {
	return ImageMap{}, nil
}

func (h *Helper) PlatformRegrade(specs []platform.RegradeSpec) (err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformRegrade",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.platform.Regrade(specs)
}

func (h *Helper) Log(args ...interface{}) {
	h.logger.Log(args...)
}
