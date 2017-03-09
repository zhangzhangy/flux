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
type ObjectSet struct {
	Source  string
	Objects map[ObjectID]Object
}

func MakeObjectSet(source string) *ObjectSet {
	return &ObjectSet{
		Source:  source,
		Objects: map[ObjectID]Object{},
	}
}

// -- unmarshaling code and diffing code for specific object and field
// types

// Container for objects, so there's something to deserialise into
type object struct {
	Object
}

// struct to embed in objects, to provide default implementation
type baseObject struct {
	Kind string `yaml:"kind"`
	Meta struct {
		Namespace string `yaml:"namespace"`
		Name      string `yaml:"name"`
	} `yaml:"metadata"`
}

func (o baseObject) ID() ObjectID {
	return ObjectID{
		Kind:      o.Kind,
		Namespace: o.Meta.Namespace,
		Name:      o.Meta.Name,
	}
}

func (obj *object) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var base baseObject
	if err := unmarshal(&base); err != nil {
		return err
	}
	if base.Meta.Namespace == "" {
		base.Meta.Namespace = "default"
	}

	switch base.Kind {
	case "Deployment":
		var dep Deployment
		if err := unmarshal(&dep); err != nil {
			return err
		}
		dep.baseObject = base
		obj.Object = &dep
		return nil
	case "Service":
		var svc Service
		if err := unmarshal(&svc); err != nil {
			return err
		}
		svc.baseObject = base
		obj.Object = &svc
		return nil
	case "Secret":
		var secret Secret
		if err := unmarshal(&secret); err != nil {
			return err
		}
		secret.baseObject = base
		obj.Object = &secret
		return nil
	case "ConfigMap":
		var config ConfigMap
		if err := unmarshal(&config); err != nil {
			return err
		}
		config.baseObject = base
		obj.Object = &config
		return nil
	case "Namespace":
		var ns Namespace
		if err := unmarshal(&ns); err != nil {
			return err
		}
		ns.baseObject = base
		obj.Object = &ns
		return nil
	}

	return errors.New("unknown object type " + base.Kind)
}

// Specific resource types are in *_resource.go
// For reference, the Kubernetes v1 types are in:
// https://github.com/kubernetes/client-go/blob/master/pkg/api/v1/types.go
