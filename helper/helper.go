package helper

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/pkg/errors"

	"github.com/weaveworks/fluxy"
	"github.com/weaveworks/fluxy/platform"
	"github.com/weaveworks/fluxy/registry"
)

type Platformer interface {
	Platform(flux.InstanceID) (platform.Platform, error)
}

type Helper struct {
	platformer Platformer
	registry   *registry.Client
	logger     log.Logger
	duration   metrics.Histogram
}

func New(
	platformer Platformer,
	registry *registry.Client,
	logger log.Logger,
	duration metrics.Histogram,
) *Helper {
	return &Helper{
		platformer: platformer,
		registry:   registry,
		logger:     logger,
		duration:   duration,
	}
}

func (h *Helper) AllServices(inst flux.InstanceID) (res []flux.ServiceID, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "AllServices",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	p, err := h.platformer.Platform(inst)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching platform for %s", inst)
	}

	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "fetching platform namespaces")
	}

	for _, namespace := range namespaces {
		ids, err := h.NamespaceServices(inst, namespace) // TODO(pb): flatten this to avoid another platform lookup!
		if err != nil {
			return nil, err
		}
		res = append(res, ids...)
	}

	return res, nil
}

func (h *Helper) NamespaceServices(inst flux.InstanceID, namespace string) (res []flux.ServiceID, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "NamespaceServices",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	p, err := h.platformer.Platform(inst)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching platform for %s", inst)
	}

	services, err := p.Services(namespace)
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
func (h *Helper) AllReleasableImagesFor(inst flux.InstanceID, serviceIDs []flux.ServiceID) (res map[flux.ServiceID][]platform.Container, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "AllReleasableImagesFor",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	p, err := h.platformer.Platform(inst)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching platform for %s", inst)
	}

	res = map[flux.ServiceID][]platform.Container{}
	for _, serviceID := range serviceIDs {
		namespace, service := serviceID.Components()
		containers, err := p.ContainersFor(namespace, service)
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

func (h *Helper) PlatformService(inst flux.InstanceID, namespace, service string) (res platform.Service, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformService",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	p, err := h.platformer.Platform(inst)
	if err != nil {
		return platform.Service{}, errors.Wrapf(err, "fetching platform for %s", inst)
	}

	return p.Service(namespace, service)
}

func (h *Helper) PlatformNamespaces(inst flux.InstanceID) (res []string, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformNamespaces",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	p, err := h.platformer.Platform(inst)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching platform for %s", inst)
	}

	return p.Namespaces()
}

func (h *Helper) PlatformContainersFor(inst flux.InstanceID, namespace, service string) (res []platform.Container, err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformContainersFor",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	p, err := h.platformer.Platform(inst)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching platform for %s", inst)
	}

	return p.ContainersFor(namespace, service)
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

func (h *Helper) PlatformRegrade(inst flux.InstanceID, specs []platform.RegradeSpec) (err error) {
	defer func(begin time.Time) {
		h.duration.With(
			"method", "PlatformRegrade",
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	p, err := h.platformer.Platform(inst)
	if err != nil {
		return errors.Wrapf(err, "fetching platform for %s", inst)
	}

	return p.Regrade(specs)
}

func (h *Helper) Log(args ...interface{}) {
	h.logger.Log(args...)
}
