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

// ==================================== TO REMOVE

func (h *Helper) AllServices() (res []flux.ServiceID, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "AllServices",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	namespaces, err := h.platform.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "fetching platform namespaces")
	}

	for _, namespace := range namespaces {
		ids, err := h.NamespaceServices(namespace)
		if err != nil {
			return nil, err
		}
		res = append(res, ids...)
	}

	return res, nil
}

func (h *Helper) NamespaceServices(namespace string) (res []flux.ServiceID, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "NamespaceServices",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	services, err := h.platform.Services(namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching platform services for namespace %q", namespace)
	}

	res = make([]flux.ServiceID, len(services))
	for i, service := range services {
		res[i] = flux.MakeServiceID(namespace, service.Name)
	}

	return res, nil
}

// AllReleasableImagesFor returns a map of service IDs to the
// containers with images that may be regraded. It leaves out any
// services that cannot have containers associated with them, e.g.,
// because there is no matching deployment.
func (h *Helper) AllReleasableImagesFor(serviceIDs []flux.ServiceID) (res map[flux.ServiceID][]platform.Container, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "AllReleasableImagesFor",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	res = map[flux.ServiceID][]platform.Container{}
	for _, serviceID := range serviceIDs {
		namespace, service := serviceID.Components()
		containers, err := h.platform.ContainersFor(namespace, service)
		if err != nil {
			switch err {
			case platform.ErrEmptySelector, platform.ErrServiceHasNoSelector, platform.ErrNoMatching, platform.ErrMultipleMatching, platform.ErrNoMatchingImages:
				continue
			default:
				return nil, errors.Wrapf(err, "fetching containers for %s", serviceID)
			}
		}
		if len(containers) <= 0 {
			continue
		}
		res[serviceID] = containers
	}
	return res, nil
}

func (h *Helper) PlatformService(namespace, service string) (res platform.Service, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformService",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.platform.Service(namespace, service)
}

func (h *Helper) PlatformNamespaces() (res []string, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformNamespaces",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.platform.Namespaces()
}

func (h *Helper) PlatformContainersFor(namespace, service string) (res []platform.Container, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformContainersFor",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.platform.ContainersFor(namespace, service)
}

func (h *Helper) RegistryGetRepository(repository string) (res *registry.Repository, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "RegistryGetRepository",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.registry.GetRepository(repository)
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
