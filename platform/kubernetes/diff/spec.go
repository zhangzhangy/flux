package diff

import (
	"fmt"
)

type PodTemplate struct {
	Metadata Meta
	Spec     PodSpec
}

type PodSpec struct {
	ImagePullSecrets []struct{ Name string }
	Volumes          []Volume
	Containers       []ContainerSpec
}

type Volume struct {
	Name   string
	Secret struct {
		SecretName string
	}
}

type ContainerSpec struct {
	Name  string
	Image string
	Args  Args
	Ports []ContainerPort
	Env   Env
}

type Args []string

func (a Args) Diff(d Differ, path string) ([]Difference, error) {
	if b, ok := d.(Args); ok {
		return diffLines([]string(a), []string(b), path)
	}
	return nil, ErrNotDiffable
}

type ContainerPort struct {
	ContainerPort int
	Name          string
}

type VolumeMount struct {
	Name      string
	MountPath string
	ReadOnly  bool
}

// Env is a bag of Name, Value pairs that are treated somewhat like a
// map.
type Env []EnvEntry

type EnvEntry struct {
	Name, Value string
}

func (a Env) Diff(d Differ, path string) ([]Difference, error) {
	if b, ok := d.(Env); ok {
		var diffs []Difference

		type entry struct {
			EnvEntry
			index int
		}

		as := map[string]entry{}
		bs := map[string]entry{}
		for i, e := range a {
			as[e.Name] = entry{e, i}
		}
		for i, e := range b {
			bs[e.Name] = entry{e, i}
		}

		for keyA, entryA := range as {
			if entryB, ok := bs[keyA]; ok {
				if entryB.Value != entryA.Value {
					diffs = append(diffs, changed{entryA.Value, entryB.Value, fmt.Sprintf("%s[%s]", path, entryA.Name)})
				}
			} else {
				diffs = append(diffs, removed{entryA.Value, fmt.Sprintf("%s[%s]", path, entryA.Name)})
			}
		}
		for keyB, entryB := range bs {
			if _, ok := as[keyB]; !ok {
				diffs = append(diffs, added{entryB.Value, fmt.Sprintf("%s[%s]", path, entryB.Name)})
			}
		}
		return diffs, nil
	}
	return nil, ErrNotDiffable
}
