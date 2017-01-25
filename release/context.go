package release

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/instance"
	"github.com/weaveworks/flux/platform"
)

type ReleaseContext struct {
	Instance           *instance.Instance
	WorkingDir         string
	Definitions        map[flux.ServiceID]map[string][]byte
	Services           []platform.Service
	Images             instance.ImageMap
	UpdatedDefinitions map[flux.ServiceID]map[string][]byte
}

func NewReleaseContext(inst *instance.Instance) *ReleaseContext {
	return &ReleaseContext{
		Instance:           inst,
		Definitions:        map[flux.ServiceID]map[string][]byte{},
		Images:             instance.ImageMap{},
		UpdatedDefinitions: map[flux.ServiceID]map[string][]byte{},
	}
}

func (rc *ReleaseContext) RepoURL() string {
	return rc.Instance.ConfigRepo().URL
}

func (rc *ReleaseContext) CloneRepo() (stderr string, err error) {
	buf := &bytes.Buffer{}
	path, err := rc.Instance.ConfigRepo().Clone(buf)
	if err != nil {
		return buf.String(), err
	}
	rc.WorkingDir = path
	return buf.String(), nil
}

func (rc *ReleaseContext) CommitAndPush(msg string) (string, error) {
	return rc.Instance.ConfigRepo().CommitAndPush(rc.WorkingDir, msg)
}

func (rc *ReleaseContext) RepoPath() string {
	return filepath.Join(rc.WorkingDir, rc.Instance.ConfigRepo().Path)
}

func (rc *ReleaseContext) Clean() {
	if rc.WorkingDir != "" {
		os.RemoveAll(rc.WorkingDir)
	}
}
