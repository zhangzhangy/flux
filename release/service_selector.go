package release

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/instance"
	"github.com/weaveworks/flux/platform"
)

type ServiceSelector interface {
	String() string
	SelectServices(*instance.Instance) ([]platform.Service, error)
	SelectDefinitions(definitionsDir string) (map[flux.ServiceID]map[string][]byte, []flux.ServiceID, error)
}

func ServiceSelectorForSpecs(inst *instance.Instance, includeSpecs []flux.ServiceSpec, exclude []flux.ServiceID) (ServiceSelector, error) {
	excludeSet := flux.ServiceIDSet{}
	excludeSet.Add(exclude)

	locked, err := lockedServices(inst)
	if err != nil {
		return nil, err
	}
	excludeSet.Add(locked)

	include := flux.ServiceIDSet{}
	for _, spec := range includeSpecs {
		if spec == flux.ServiceSpecAll {
			// If one of the specs is '<all>' we can ignore the rest.
			return AllServicesExcept(excludeSet), nil
		}
		serviceID, err := flux.ParseServiceID(string(spec))
		if err != nil {
			return nil, errors.Wrapf(err, "parsing service ID from params %q", spec)
		}
		include.Add([]flux.ServiceID{serviceID})
	}
	return ExactlyTheseServices(include.Without(excludeSet)), nil
}

type funcServiceQuery struct {
	text              string
	selectServices    func(inst *instance.Instance) ([]platform.Service, error)
	selectDefinitions func(string) (map[flux.ServiceID][]byte, error)
}

func (f funcServiceQuery) String() string {
	return f.text
}

func (f funcServiceQuery) SelectServices(inst *instance.Instance) ([]platform.Service, error) {
	return f.selectServices(inst)
}

func (f funcServiceQuery) SelectDefinitions(path string) (map[flux.ServiceID]map[string][]byte, error) {
	return f.selectDefinitions(path)
}

func ExactlyTheseServices(include flux.ServiceIDSet) ServiceSelector {
	var (
		idText  []string
		idSlice []flux.ServiceID
	)
	text := "no services"
	if len(include) > 0 {
		for id := range include {
			idText = append(idText, string(id))
			idSlice = append(idSlice, id)
		}
		text = strings.Join(idText, ", ")
	}
	return funcServiceQuery{
		text: text,
		selectServices: func(h *instance.Instance) ([]platform.Service, error) {
			return h.GetServices(idSlice)
		},
		selectDefinitions: func(path string) (map[flux.ServiceID]map[string][]byte, []flux.ServiceID, error) {
			all, _, err := AllServicesExcept.SelectDefinitions(path)
			if err != nil {
				return nil, nil, err
			}

			var skipped []flux.ServiceID
			definitions := map[flux.ServiceID][]byte{}
			for id := range include {
				def, ok := all[id]
				if !ok {
					skipped = append(skipped, id)
					continue
				}
				definitions[id] = def
			}
			return definitions, skipped, nil
		},
	}
}

func AllServicesExcept(exclude flux.ServiceIDSet) ServiceSelector {
	text := "all services"
	if len(exclude) > 0 {
		var idText []string
		for id := range exclude {
			idText = append(idText, string(id))
		}
		text += fmt.Sprintf(" (except: %s)", strings.Join(idText, ", "))
	}
	return funcServiceQuery{
		text: text,
		selectServices: func(h *instance.Instance) ([]platform.Service, error) {
			return h.GetAllServicesExcept("", exclude)
		},
		selectDefinitions: func(path string) (map[flux.ServiceID]map[string][]byte, []flux.ServiceID, error) {
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				return "", fmt.Errorf("the resource path (%s) is not valid", path)
			}

			allFiles, err := kubernetes.DefinedServices(path)
			if err != nil {
				return nil, nil, err
			}

			definitions := map[flux.ServiceID]map[string][]byte{}
			for id, files := range all {
				if exclude.Contains(id) {
					continue
				}

				if len(files) > 1 {
					return nil, nil, fmt.Errorf("multiple resource definition files found for %s: %s", id, strings.Join(files, ", "))
				}
				if len(files) <= 0 {
					continue
				}

				def, err := ioutil.ReadFile(files[0]) // TODO(mb) not multi-doc safe
				if err != nil {
					return nil, nil, err
				}
				definitions[id] = map[string][]byte{files[0]: def}
			}
			return definitions, nil, nil
		},
	}
}

func lockedServices(inst *instance.Instance) ([]flux.ServiceID, error) {
	config, err := inst.GetConfig()
	if err != nil {
		return nil, err
	}

	ids := []flux.ServiceID{}
	for id, s := range config.Services {
		if s.Locked {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
