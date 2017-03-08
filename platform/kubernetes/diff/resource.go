package diff

import (
	"errors"
)

// https://kubernetes.io/docs/user-guide/identifiers/
// Objects are unique by {Namespace, Kind, Name}
type ObjectID struct {
	Namespace string
	Kind      string
	Name      string
}

type Object interface {
	ID() ObjectID
}

// ObjectSet is a set of several objects which can be diffed
// collectively.
type ObjectSet map[ObjectID]Object

// -- unmarshaling code and diffing code for specific object and field
// types

// Container for objects, so there's something to deserialise into
type object struct {
	Object
}

// struct to embed in objects, to provide default implementation
type baseObject struct {
	ObjectID
}

func (o baseObject) ID() ObjectID {
	return o.ObjectID
}

func (obj *object) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var objID struct {
		Kind string `yaml:"kind"`
		Meta struct {
			Namespace string `yaml:"namespace"`
			Name      string `yaml:"name"`
		} `yaml:"metadata"`
	}
	if err := unmarshal(&objID); err != nil {
		return err
	}

	id := ObjectID{
		Kind:      objID.Kind,
		Name:      objID.Meta.Name,
		Namespace: objID.Meta.Namespace,
	}
	if id.Namespace == "" {
		id.Namespace = "default"
	}

	switch id.Kind {
	case "Deployment":
		var dep Deployment
		if err := unmarshal(&dep); err != nil {
			return err
		}
		dep.baseObject = baseObject{id}
		obj.Object = &dep
		return nil
	case "Service":
		var svc Service
		if err := unmarshal(&svc); err != nil {
			return err
		}
		svc.baseObject = baseObject{id}
		obj.Object = &svc
		return nil
	}

	return errors.New("unknown object type " + objID.Kind)
}

// Specific resource types are in *_resource.go
// For reference, the Kubernetes v1 types are in:
// https://github.com/kubernetes/client-go/blob/master/pkg/api/v1/types.go
