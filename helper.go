package flux

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/pkg/errors"

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

func NewHelper(
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

// ImagesFor gets the image metadata for the images used in the
// servics given.
func (h *Helper) ImagesFor(services []Service) (res map[ServiceID][]Container, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "ImagesFor",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	images := map[string][]ImageDescription{}
	for _, service := range services {
		for _, container := range service.Containers {
			image := container.Current
			images[image.Repository] = make([]ImageDescription)
		}
	}

	var errs []error
	for repository, _ := range images {
		registryRepo, err := h.RegistryGetRepository(repository)
		if err != nil {
			append(errs, err)
		}
		images[repository] = makeImageDescriptions(registryRepo)
	}

	res = map[ServiceID][]Container{}
	for _, service := range services {
		res[service.ID] = makeContainersWithImages(service, images)
	}
	return res, nil
}

func makeImageDescriptions(repo registry.Repository) []ImageDescription {
	res := []ImageDescription{}
	for _, image := range repo.Images {
		res = append(res, ImageDescription{
			ID:        MakeImageID(image.Registry, image.Name, image.Tag),
			CreatedAt: image.CreatedAt,
		})
	}
}

// PlatformServices asks the platform for a list of the services,
// either those running in the namespace given, or if it is empty,
// those running in all namespaces.
func (h *Helper) PlatformServices(namespace string) (res []platform.Service, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformService",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.platform.Services(namespace)
}

func (h *Helper) PlatformService(serviceID flux.ServiceID) (platform.Service, error) {
	ns, s := serviceID.Components()
	return h.Service(ns, s)
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

func (h *Helper) RegistryGetRepository(repository string) (res *registry.Repository, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "RegistryGetRepository",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.registry.GetRepository(repository)
}

func (h *Helper) PlatformRegrade(namespace, serviceName string, newDefinition []byte) (err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformRegrade",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return h.platform.Regrade(namespace, serviceName, newDefinition)
}

func (h *Helper) Log(args ...interface{}) {
	h.logger.Log(args...)
}
