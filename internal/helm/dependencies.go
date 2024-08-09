package helm

import (
	"io"

	"helm.sh/helm/v3/pkg/downloader"
)

// UpdateDependencies fetches the chart dependencies.
// See https://github.com/helm/helm/blob/v3.15.3/cmd/helm/dependency_update.go
func UpdateDependencies(cfg Config, out io.Writer) error {
	man := &downloader.Manager{
		Out:              out,
		ChartPath:        cfg.ChartPath,
		Keyring:          cfg.Keyring,
		SkipUpdate:       true,
		Getters:          cfg.Getters,
		RegistryClient:   cfg.RegistryClient,
		RepositoryConfig: cfg.Settings.RepositoryConfig,
		RepositoryCache:  cfg.Settings.RepositoryCache,
		Debug:            cfg.Settings.Debug,
	}
	if cfg.Verify {
		man.Verify = downloader.VerifyAlways
	}

	return man.Update()
}
