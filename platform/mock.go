package platform

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/weaveworks/flux"
)

type MockPlatform struct {
	AllServicesArgTest func(string, flux.ServiceIDSet) error
	AllServicesAnswer  []Service
	AllServicesError   error

	SomeServicesArgTest func([]flux.ServiceID) error
	SomeServicesAnswer  []Service
	SomeServicesError   error

	ApplyArgTest func([]ServiceDefinition) error
	ApplyError   error

	PingError error

	VersionAnswer string
	VersionError  error

	SyncArgTest func(SyncDef) error
	SyncError   error
}

func (p *MockPlatform) AllServices(ns string, ss flux.ServiceIDSet) ([]Service, error) {
	if p.AllServicesArgTest != nil {
		if err := p.AllServicesArgTest(ns, ss); err != nil {
			return nil, err
		}
	}
	return p.AllServicesAnswer, p.AllServicesError
}

func (p *MockPlatform) SomeServices(ss []flux.ServiceID) ([]Service, error) {
	if p.SomeServicesArgTest != nil {
		if err := p.SomeServicesArgTest(ss); err != nil {
			return nil, err
		}
	}
	return p.SomeServicesAnswer, p.SomeServicesError
}

func (p *MockPlatform) Apply(defs []ServiceDefinition) error {
	if p.ApplyArgTest != nil {
		if err := p.ApplyArgTest(defs); err != nil {
			return err
		}
	}
	return p.ApplyError
}

func (p *MockPlatform) Ping() error {
	return p.PingError
}

func (p *MockPlatform) Version() (string, error) {
	return p.VersionAnswer, p.VersionError
}

func (p *MockPlatform) Sync(def SyncDef) error {
	if p.SyncArgTest != nil {
		if err := p.SyncArgTest(def); err != nil {
			return err
		}
	}
	return p.SyncError
}

// -- battery of tests for a platform mechanism

func PlatformTestBattery(t *testing.T, wrap func(mock Platform) Platform) {
	// set up
	namespace := "space-of-names"
	serviceID := flux.ServiceID(namespace + "/service")
	serviceList := []flux.ServiceID{serviceID}
	services := flux.ServiceIDSet{}
	services.Add(serviceList)

	expectedDefs := []ServiceDefinition{
		{
			ServiceID:     serviceID,
			NewDefinition: []byte("imagine a definition here"),
		},
	}

	serviceAnswer := []Service{
		Service{
			ID:       flux.ServiceID("foobar/hello"),
			IP:       "10.32.1.45",
			Metadata: map[string]string{},
			Status:   "ok",
			Containers: ContainersOrExcuse{
				Containers: []Container{
					Container{
						Name:  "frobnicator",
						Image: "quay.io/example.com/frob:v0.4.5",
					},
				},
			},
		},
		Service{},
	}

	expectedSyncDef := SyncDef{
		Actions: map[ResourceID]SyncAction{
			ResourceID("deployment/foo/bar"): SyncAction{
				Delete: []byte("delete this"),
			},
			ResourceID("service/foo/bar"): SyncAction{
				Apply:  []byte("apply this"),
				Create: []byte("create this"),
			},
		},
	}

	mock := &MockPlatform{
		AllServicesArgTest: func(ns string, ss flux.ServiceIDSet) error {
			if !(ns == namespace &&
				ss.Contains(serviceID)) {
				return fmt.Errorf("did not get expected args, got %q, %+v", ns, ss)
			}
			return nil
		},
		AllServicesAnswer: serviceAnswer,

		SomeServicesArgTest: func(ss []flux.ServiceID) error {
			if !reflect.DeepEqual(ss, serviceList) {
				return fmt.Errorf("did not get expected args, got %+v", ss)
			}
			return nil
		},
		SomeServicesAnswer: serviceAnswer,

		ApplyArgTest: func(defs []ServiceDefinition) error {
			if !reflect.DeepEqual(expectedDefs, defs) {
				return fmt.Errorf("did not get expected args, got %+v", defs)
			}
			return nil
		},
		ApplyError: nil,

		SyncArgTest: func(def SyncDef) error {
			if !reflect.DeepEqual(expectedSyncDef, def) {
				return fmt.Errorf("did not get expected sync def, got %+v", def)
			}
			return nil
		},
		SyncError: nil,
	}

	// OK, here we go
	client := wrap(mock)

	if err := client.Ping(); err != nil {
		t.Fatal(err)
	}

	ss, err := client.AllServices(namespace, services)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(ss, mock.AllServicesAnswer) {
		t.Error(fmt.Errorf("expected %d result(s), got %+v", len(mock.AllServicesAnswer), ss))
	}
	mock.AllServicesError = fmt.Errorf("all services query failure")
	ss, err = client.AllServices(namespace, services)
	if err == nil {
		t.Error("expected error, got nil")
	}

	ss, err = client.SomeServices(serviceList)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(ss, mock.SomeServicesAnswer) {
		t.Error(fmt.Errorf("expected %d result(s), got %+v", len(mock.SomeServicesAnswer), ss))
	}
	mock.SomeServicesError = fmt.Errorf("fail for some reason")
	ss, err = client.SomeServices(serviceList)
	if err == nil {
		t.Error("expected error, got nil")
	}

	err = client.Apply(expectedDefs)
	if err != nil {
		t.Error(err)
	}

	applyErrors := ApplyError{
		serviceID: fmt.Errorf("it just failed"),
	}
	mock.ApplyError = applyErrors
	err = client.Apply(expectedDefs)
	if !reflect.DeepEqual(err, applyErrors) {
		t.Errorf("expected ApplyError, got %#v", err)
	}

	err = client.Sync(expectedSyncDef)
	if err != nil {
		t.Error(err)
	}

	syncErrors := SyncError{
		ResourceID("deployment/foo/bar"): errors.New("delete failed for this"),
	}
	mock.SyncError = syncErrors
	err = client.Sync(expectedSyncDef)
	if !reflect.DeepEqual(err, syncErrors) {
		t.Errorf("expected SyncError, got %+v", err)
	}
}
