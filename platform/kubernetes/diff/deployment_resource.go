package diff

type Deployment struct {
	baseObject
	Spec DeploymentSpec
}

type DeploymentSpec struct {
	Replicas int
	Template PodTemplate
}

type PodTemplate struct {
	Metadata Meta
	Spec     PodSpec
}

type Meta struct {
	Labels      map[string]string
	Annotations map[string]string
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

type Env []EnvEntry

type EnvEntry struct {
	Name, Value string
}

// TODO implement Env diff -- ignore order in which they are defined,
// since this doesn't affect the running object.
