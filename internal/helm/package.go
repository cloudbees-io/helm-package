package helm

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
)

// Packages a given Helm chart.
// See https://github.com/helm/helm/blob/v3.15.3/cmd/helm/package.go
func PackageChart(cfg Config, out io.Writer) (destDir string, err error) {
	client := action.NewPackage()

	client.Sign = cfg.Sign
	client.Key = cfg.SignKey
	client.Keyring = cfg.Keyring
	client.PassphraseFile = cfg.PassphraseFile
	client.Version = cfg.Version
	client.AppVersion = cfg.AppVersion
	client.Destination = cfg.Destination
	client.RepositoryConfig = cfg.Settings.RepositoryConfig
	client.RepositoryCache = cfg.Settings.RepositoryCache

	if cfg.Sign {
		if cfg.SignKey == "" {
			return "", errors.New("sign-key is required for signing a package")
		}
		if cfg.Keyring == "" {
			return "", errors.New("keyring is required for signing a package")
		}
	}

	path, err := filepath.Abs(cfg.ChartPath)
	if err != nil {
		return "", fmt.Errorf("chart: %w", err)
	}

	if _, err := os.Stat(cfg.ChartPath); err != nil {
		return "", fmt.Errorf("chart: %w", err)
	}

	if cfg.EmbedValues != "" {
		valuesFile := filepath.Join(cfg.ChartPath, "values.yaml")

		valuesBackup, err := os.ReadFile(valuesFile)
		if err != nil {
			return "", fmt.Errorf("reading values.yaml: %w", err)
		}

		vals, err := mergeChartValues(valuesFile, cfg.EmbedValues, cfg.Getters)
		if err != nil {
			return "", err
		}

		b, err := yaml.Marshal(vals)
		if err != nil {
			return "", fmt.Errorf("marshalling merged chart values: %w", err)
		}

		err = os.WriteFile(valuesFile, b, 0640)
		if err != nil {
			return "", fmt.Errorf("writing merged chart values: %w", err)
		}

		defer func() {
			// Restore values.yaml
			e := os.WriteFile(valuesFile, valuesBackup, 0640)
			if e != nil && err == nil {
				err = e
			}
		}()
	}

	dest, err := client.Run(path, nil)
	if err != nil {
		return "", err
	}

	return dest, err
}
