package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/client-go/util/homedir"
)

// Config defines the configuration options.
type Config struct {
	ChartPath      string
	Version        string
	AppVersion     string
	EmbedValues    string
	Verify         bool
	Sign           bool
	SignKey        string
	Keyring        string
	PassphraseFile string
	Destination    string
	Settings       *cli.EnvSettings
	Getters        getter.Providers
	RegistryClient *registry.Client
	RegistryConfig string
}

// NewConfig creates new configuration options with default values.
func NewConfig() Config {
	return Config{
		Settings: cli.New(),
		Keyring:  defaultKeyring(),
	}
}

func (c *Config) Complete() error {
	registryClient, err := newRegistryClient(c.Settings)
	if err != nil {
		return fmt.Errorf("new registry client: %w", err)
	}

	c.RegistryClient = registryClient
	c.Getters = All(c.Settings, c.RegistryConfig)

	return nil
}

func defaultKeyring() string {
	if v, ok := os.LookupEnv("GNUPGHOME"); ok {
		return filepath.Join(v, "pubring.gpg")
	}

	return filepath.Join(homedir.HomeDir(), ".gnupg", "pubring.gpg")
}

func newRegistryClient(settings *cli.EnvSettings) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	}

	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return registryClient, nil
}
