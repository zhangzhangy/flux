package diff

import (
	"errors"
	"fmt"
	"io"
)

// For reference:
// https://github.com/kubernetes/client-go/blob/master/pkg/api/v1/types.go#L2641

type Service struct {
	baseObject
	Spec ServiceSpec `yaml:"spec"`
}

type ServiceSpec struct {
	Type     string        `yaml:"type"`
	Ports    []ServicePort `yaml:"ports"`
	Selector Selector      `yaml:"selector"`
}

type Selector map[string]string

func (s Selector) Diff(other Differ, path string) ([]Difference, error) {
	if s1, ok := other.(Selector); ok {
		return diffMap(s, s1, path)
	}
	return nil, errors.New("not comparable to selector")
}

func diffMap(a, b map[string]string, path string) ([]Difference, error) {
	diff := mapDifference{
		path:      path,
		OnlyA:     map[string]string{},
		OnlyB:     map[string]string{},
		Different: map[string]valueDifference{},
	}

	for keyA, valA := range a {
		if valB, ok := b[keyA]; ok {
			if valA != valB {
				diff.Different[keyA] = valueDifference{valA, valB, path + fmt.Sprintf(`[%q]`, keyA)}
			}
		} else {
			diff.OnlyA[keyA] = valA
		}
	}
	for keyB, valB := range b {
		if _, ok := a[keyB]; !ok {
			diff.OnlyB[keyB] = valB
		}
	}

	if len(diff.OnlyA) > 0 || len(diff.OnlyB) > 0 || len(diff.Different) > 0 {
		return []Difference{diff}, nil
	}
	return nil, nil
}

type mapDifference struct {
	path string

	OnlyA     map[string]string
	OnlyB     map[string]string
	Different map[string]valueDifference
}

func (d mapDifference) Summarise(out io.Writer) {
	fmt.Fprintf(out, "%s: map difference", d.path)
}

type ServicePort struct {
	Name       string `yaml:"name"`
	Protocol   string `yaml:"protocol"`
	Port       int32  `yaml:"port"`
	TargetPort string `yaml:"targetPort"`
	NodePort   int32  `yaml:"nodePort"`
}
