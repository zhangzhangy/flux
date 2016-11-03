package flux

// Instance configuration, mutated via `fluxctl config`. It can be
// supplied as YAML (hence YAML annotations) and is transported as
// JSON (hence JSON annotations).

const secretReplacement = "******"

type GitConfig struct {
	URL       string `json:"URL" yaml:"URL"`
	Path      string `json:"path" yaml:"path"`
	Branch    string `json:"branch" yaml:"branch"`
	Key       string `json:"key" yaml:"key"`
	PublicKey string `json:"publicKey" yaml:"publicKey"`
}

type SlackConfig struct {
	HookURL  string `json:"hookURL" yaml:"hookURL"`
	Username string `json:"username" yaml:"username"`
}

type RegistryConfig struct {
	// Map of index host to Basic auth string (base64 encoded
	// username:password), to make it easy to copypasta from docker
	// config.
	Auths map[string]struct {
		Auth string `json:"auth" yaml:"auth"`
	} `json:"auths" yaml:"auths"`
}

type InstanceConfig struct {
	Git      GitConfig      `json:"git" yaml:"git"`
	Slack    SlackConfig    `json:"slack" yaml:"slack"`
	Registry RegistryConfig `json:"registry" yaml:"registry"`
}

type ConfigUpdate struct {
	GenerateKey bool           `json:"generateKey"`
	Config      InstanceConfig `json:"config"`
}

func (config InstanceConfig) HideSecrets() InstanceConfig {
	for _, auth := range config.Registry.Auths {
		auth.Auth = secretReplacement
	}
	if config.Git.Key != "" {
		config.Git.Key = secretReplacement
	}
	return config
}
