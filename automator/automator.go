package automator

import (
	"sync"

	"github.com/weaveworks/fluxy"
)

const (
	automationEnabled  = "Automation enabled."
	automationDisabled = "Automation disabled."

	HardwiredInstance = "DEFAULT"
)

// Automator orchestrates continuous deployment for specific services.
type Automator struct {
	cfg Config
}

// New creates a new automator.
func New(cfg Config) (*Automator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	auto := &Automator{
		cfg: cfg,
	}
	return auto, nil
}

func (a *Automater) setAutomation(service ServiceID, automation bool) error {
	instanceConfig, err := a.cfg.Config.Get(HardwiredInstance)
	if err != nil {
		return err
	}
	if _, found := instanceConfig.Services[service]; found {
		instanceConfig.Services[service].Automated = automation
	} else {
		instanceConfig.Services[service] = config.ServiceConfig{
			Automation: true,
		}
	}
	if err = a.cfg.Config.Set(HardwiredInstance, instanceConfig); err != nil {
		return err
	}
}

// Automate turns on automated (continuous) deployment for the named service.
// This call always succeeds; if the named service cannot be automated for some
// reason, that will be detected and happen autonomously.
func (a *Automator) Automate(namespace, serviceName string) error {
	a.cfg.History.LogEvent(namespace, serviceName, automationEnabled)
	return a.setAutomation(flux.MakeServiceID(namespace, serviceName), true)
}

// Deautomate turns off automated (continuous) deployment for the named service.
// This is more of a signal; it may take some time for the service to be
// properly deautomated.
func (a *Automator) Deautomate(namespace, serviceName string) error {
	a.cfg.History.LogEvent(namespace, serviceName, automationDisabled)
	return a.setAutomation(flux.MakeServiceID(namespace, serviceName), false)
}

func (a *Automator) automate(namespace, serviceName string) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	ns := namespacedService{namespace, serviceName}
	if _, ok := a.active[ns]; ok {
		return nil
	}

	onDelete := func() { a.deleteCallback(namespace, serviceName) }
	svcLogFunc := makeServiceLogFunc(a.cfg.History, namespace, serviceName)
	s := newSvc(namespace, serviceName, svcLogFunc, onDelete, a.cfg)
	a.active[ns] = s

	a.cfg.History.LogEvent(namespace, serviceName, automationEnabled)
	return nil
}

// Deautomate turns off automated (continuous) deployment for the named service.
// This is more of a signal; it may take some time for the service to be
// properly deautomated.
func (a *Automator) Deautomate(namespace, serviceName string) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	ns := namespacedService{namespace, serviceName}
	s, ok := a.active[ns]
	if !ok {
		return nil
	}

	// We signal delete rather than actually deleting anything here,
	// to make sure svc termination follows a single code path.
	s.signalDelete()

	return nil
}

// IsAutomated checks if a given service has automation enabled.
func (a *Automator) IsAutomated(namespace, serviceName string) bool {
	if a == nil {
		return false
	}
	a.mtx.RLock()
	_, ok := a.active[namespacedService{namespace, serviceName}]
	a.mtx.RUnlock()
	return ok
}

// deleteCallback is invoked by a svc when it shuts down. A svc may terminate
// itself, and so needs this as a form of accounting.
func (a *Automator) deleteCallback(namespace, serviceName string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	ns := namespacedService{namespace, serviceName}
	delete(a.active, ns)
}

type namespacedService struct {
	namespace string
	service   string
}
