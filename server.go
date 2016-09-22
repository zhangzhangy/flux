package flux

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/pkg/errors"

	"github.com/weaveworks/fluxy/history"
	"github.com/weaveworks/fluxy/platform/kubernetes"
	"github.com/weaveworks/fluxy/registry"
)

type server struct {
	helper      *Helper
	releaser    ReleaseJobReadPusher
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

func NewServer(
	platform *kubernetes.Cluster,
	registry *registry.Client,
	releaser ReleaseJobReadPusher,
	automator Automator,
	history history.EventReader,
	logger log.Logger,
	metrics Metrics,
	helperDuration metrics.Histogram,
) Service {
	return &server{
		helper:      NewHelper(platform, registry, logger, helperDuration),
		releaser:    releaser,
		automator:   automator,
		history:     history,
		maxPlatform: make(chan struct{}, 8),
		metrics:     metrics,
	}
}

func (s *server) ListServices(namespace string) (res []ServiceStatus, err error) {
	defer func(begin time.Time) {
		s.metrics.ListServicesDuration.With(
			"namespace", namespace,
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	services, err = s.helper.PlatformServices(namespace)

	if err != nil {
		return nil, errors.Wrapf(err, "fetching services for namespace %s on the platform", namespace)
	}

	for _, service := range services {
		ns, s := service.ID.Components()
		status := ServiceStatus{
			ID:         service.ID,
			Containers: makeServiceContainers(service.Containers),
			Status:     service.Status,
			Automated:  s.automator.IsAutomated(ns, s),
		}
		res = append(res, status)
	}
	return res, nil
}

func makeServiceContainers(cs []platform.Container) (res []Container) {
	res = make([]Container, len(cs))
	for i, c := range cs {
		res[i] = Container{
			Name:    c.Name,
			Current: ImageDescription{ID: ParseImageID(c.Image)},
		}
	}
}

func (s *server) ListImages(spec ServiceSpec) (res []ImageStatus, err error) {
	defer func(begin time.Time) {
		s.metrics.ListImagesDuration.With(
			"service_spec", fmt.Sprint(spec),
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	services, err := func() ([]ServiceID, error) {
		if spec == ServiceSpecAll {
			return s.helper.PlatformServices()
		}
		s, err := s.helper.PlatformOneService(ParseServiceID(string(spec)))
		return []platform.Service{s}, err
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "fetching services from platform")
	}

	serviceContainers, err := s.helper.ContainersFor(services)
	if err != nil {
		return nil, errors.Wrap(err, "fetching image metadata for service containers")
	}

	for i, service := range services {
		res = append(res, ImageStatus{
			ID:         service.ID,
			Containers: serviceContainers[i],
		})
	}
	return res, nil
}

func (s *server) History(spec ServiceSpec) (res []HistoryEntry, err error) {
	defer func(begin time.Time) {
		s.metrics.HistoryDuration.With(
			"service_spec", fmt.Sprint(spec),
			"success", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	var events []history.Event
	if spec == ServiceSpecAll {
		namespaces, err := s.helper.PlatformNamespaces()
		if err != nil {
			return nil, errors.Wrap(err, "fetching platform namespaces")
		}

		for _, namespace := range namespaces {
			ev, err := s.history.AllEvents(namespace)
			if err != nil {
				return nil, errors.Wrapf(err, "fetching all history events for namespace %s", namespace)
			}

			events = append(events, ev...)
		}
	} else {
		id, err := ParseServiceID(string(spec))
		if err != nil {
			return nil, errors.Wrapf(err, "parsing service ID from spec %s", spec)
		}

		namespace, service := id.Components()
		ev, err := s.history.EventsForService(namespace, service)
		if err != nil {
			return nil, errors.Wrapf(err, "fetching history events for %s", id)
		}

		events = append(events, ev...)
	}

	res = make([]HistoryEntry, len(events))
	for i, event := range events {
		res[i] = HistoryEntry{
			Stamp: event.Stamp,
			Type:  "v0",
			Data:  fmt.Sprintf("%s: %s", event.Service, event.Msg),
		}
	}

	return res, nil
}

func (s *server) Automate(service ServiceID) error {
	ns, svc := service.Components()
	return s.automator.Automate(ns, svc)
}

func (s *server) Deautomate(service ServiceID) error {
	ns, svc := service.Components()
	return s.automator.Deautomate(ns, svc)
}

func (s *server) PostRelease(spec ReleaseJobSpec) (ReleaseID, error) {
	return s.releaser.PutJob(spec)
}

func (s *server) GetRelease(id ReleaseID) (ReleaseJob, error) {
	return s.releaser.GetJob(id)
}

type compositeError []error

func (e compositeError) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}
