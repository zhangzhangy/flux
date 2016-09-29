package server

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/pkg/errors"

	"github.com/weaveworks/fluxy"
	"github.com/weaveworks/fluxy/helper"
	"github.com/weaveworks/fluxy/history"
	"github.com/weaveworks/fluxy/platform"
	"github.com/weaveworks/fluxy/platform/kubernetes"
	"github.com/weaveworks/fluxy/registry"
)

type server struct {
	helper      *helper.Helper
	releaser    flux.ReleaseJobReadPusher
	automator   Automator
	history     history.EventReader
	maxPlatform chan struct{} // semaphore for concurrent calls to the platform
	metrics     Metrics
}

type Automator interface {
	Automate(namespace, service string) error
	Deautomate(namespace, service string) error
	IsAutomated(namespace, service string) bool
}

type Metrics struct {
	ListServicesDuration metrics.Histogram
	ListImagesDuration   metrics.Histogram
	HistoryDuration      metrics.Histogram
}

func New(
	platform *kubernetes.Cluster,
	registry *registry.Client,
	releaser flux.ReleaseJobReadPusher,
	automator Automator,
	history history.EventReader,
	logger log.Logger,
	metrics Metrics,
	helperDuration metrics.Histogram,
) flux.Service {
	return &server{
		helper:      helper.New(platform, registry, logger, helperDuration),
		releaser:    releaser,
		automator:   automator,
		history:     history,
		maxPlatform: make(chan struct{}, 8),
		metrics:     metrics,
	}
}

// The server methods are deliberately awkward, cobbled together from existing
// platform and registry APIs. I want to avoid changing those components until I
// get something working. There's also a lot of code duplication here for the
// same reason: let's not add abstraction until it's merged, or nearly so, and
// it's clear where the abstraction should exist.

func (s *server) ListServices(namespace string) (res []flux.ServiceStatus, err error) {
	defer func(begin time.Time) {
		s.metrics.ListServicesDuration.With(
			"namespace", namespace,
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	services, err := s.helper.GetAllServices(namespace)
	if err != nil {
		return nil, errors.Wrap(err, "getting services from platform")
	}

	for _, service := range services {
		namespace, serviceName := service.ID.Components()
		res = append(res, flux.ServiceStatus{
			ID:         service.ID,
			Containers: containers2containers(service.ContainersOrNil()),
			Status:     service.Status,
			Automated:  s.automator.IsAutomated(namespace, serviceName),
		})
	}
	return res, nil
}

func containers2containers(cs []platform.Container) []flux.Container {
	res := make([]flux.Container, len(cs))
	for i, c := range cs {
		res[i] = flux.Container{
			Name: c.Name,
			Current: flux.ImageDescription{
				ID: flux.ParseImageID(c.Image),
			},
		}
	}
	return res
}

func (s *server) ListImages(spec flux.ServiceSpec) (res []flux.ImageStatus, err error) {
	defer func(begin time.Time) {
		s.metrics.ListImagesDuration.With(
			"service_spec", fmt.Sprint(spec),
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	var services []platform.Service
	if spec == flux.ServiceSpecAll {
		services, err = s.helper.GetAllServices("")
	} else {
		id, err := spec.AsID()
		if err != nil {
			return nil, errors.Wrap(err, "treating service spec as ID")
		}
		services, err = s.helper.GetServices([]flux.ServiceID{id})
	}

	images, err := s.helper.CollectAvailableImages(services)
	if err != nil {
		return nil, errors.Wrap(err, "getting images for services")
	}

	for _, service := range services {
		containers := containersWithAvailable(service, images)
		res = append(res, flux.ImageStatus{
			ID:         service.ID,
			Containers: containers,
		})
	}

	return res, nil
}

func containersWithAvailable(service platform.Service, images helper.ImageMap) (res []flux.Container) {
	for _, c := range service.ContainersOrNil() {
		id := flux.ParseImageID(c.Image)
		repo := id.Repository()
		available := images[repo]
		res = append(res, flux.Container{
			Name: c.Name,
			Current: flux.ImageDescription{
				ID: id,
			},
			Available: available,
		})
	}
	return res
}

func (s *server) History(spec flux.ServiceSpec) (res []flux.HistoryEntry, err error) {
	defer func(begin time.Time) {
		s.metrics.HistoryDuration.With(
			"service_spec", fmt.Sprint(spec),
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	var events []history.Event
	if spec == flux.ServiceSpecAll {
		events, err = s.history.AllEvents()
		if err != nil {
			return nil, errors.Wrap(err, "fetching all history events")
		}
	} else {
		id, err := flux.ParseServiceID(string(spec))
		if err != nil {
			return nil, errors.Wrapf(err, "parsing service ID from spec %s", spec)
		}

		namespace, service := id.Components()
		events, err = s.history.EventsForService(namespace, service)
		if err != nil {
			return nil, errors.Wrapf(err, "fetching history events for %s", id)
		}
	}

	res = make([]flux.HistoryEntry, len(events))
	for i, event := range events {
		res[i] = flux.HistoryEntry{
			Stamp: event.Stamp,
			Type:  "v0",
			Data:  fmt.Sprintf("%s: %s", event.Service, event.Msg),
		}
	}

	return res, nil
}

func (s *server) Automate(service flux.ServiceID) error {
	ns, svc := service.Components()
	return s.automator.Automate(ns, svc)
}

func (s *server) Deautomate(service flux.ServiceID) error {
	ns, svc := service.Components()
	return s.automator.Deautomate(ns, svc)
}

func (s *server) PostRelease(spec flux.ReleaseJobSpec) (flux.ReleaseID, error) {
	return s.releaser.PutJob(spec)
}

func (s *server) GetRelease(id flux.ReleaseID) (flux.ReleaseJob, error) {
	return s.releaser.GetJob(id)
}
