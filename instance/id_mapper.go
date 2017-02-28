package instance

import (
	"github.com/weaveworks/flux"
)

// IDMapper converts internal instance ids into external instance ids. Maybe
// this interface is overkill, we'll see what the users client looks like.
type IDMapper interface {
	ConvertInternalInstanceIDToExternal(flux.InstanceID) (string, error)
}

type IDMapperFunc func(flux.InstanceID) (string, error)

func (f IDMapperFunc) ConvertInternalInstanceIDToExternal(inst flux.InstanceID) (string, error) {
	return f(inst)
}

// IdentityIDMapper is a noop ID Mapper which just returns the internal
// instance id. Useful in testing. Disastrous in prod.
var IdentityIDMapper = IDMapperFunc(func(inst flux.InstanceID) (string, error) {
	if inst == flux.DefaultInstanceID {
		return "", nil
	}
	return string(inst), nil
})
