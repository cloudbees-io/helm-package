package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/cloudbees-io/helm-package/internal/helm"
)

const envVarPrefix = "CBHELMPKG_"

func Execute(out io.Writer) error {
	cfg := helm.NewConfig()

	cmd := &cobra.Command{
		Use:   "cbhelmpkg CHARTPATH",
		Short: "Package a given chart",
		Long:  "Package a given chart",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			err := cfg.Complete()
			if err != nil {
				return err
			}

			actionOutDir := os.Getenv("CLOUDBEES_OUTPUTS")
			runnerTempDir := os.Getenv("RUNNER_TEMP")
			if actionOutDir == "" || runnerTempDir == "" {
				msg := "env vars CLOUDBEES_OUTPUTS and RUNNER_TEMP must be specified" +
					" - this binary is built to run within a CloudBees Action"
				return errors.New(msg)
			}

			if cfg.Destination == "" {
				runnerTempDir = filepath.Clean(runnerTempDir)

				cfg.Destination, err = os.MkdirTemp(runnerTempDir, "chart-")
				if err != nil {
					return fmt.Errorf("create temp chart destination dir: %w", err)
				}
			} else {
				cfg.Destination = filepath.Clean(cfg.Destination)
			}

			cfg.ChartPath = filepath.Clean(args[0])

			chart, err := loader.Load(cfg.ChartPath)
			if err != nil {
				return fmt.Errorf("load chart: %w", err)
			}

			if cfg.AppVersion == "" {
				if cfg.Version == "" {
					if chart.AppVersion() == "" {
						cfg.AppVersion = chart.Metadata.Version
					} else {
						cfg.AppVersion = chart.AppVersion()
					}
				} else {
					cfg.AppVersion = cfg.Version
				}
			}

			if cfg.Version == "" {
				cfg.Version = chart.Metadata.Version
			}

			err = helm.UpdateDependencies(cfg, out)
			if err != nil {
				return fmt.Errorf("build chart dependencies: %w", err)
			}

			dest, err := helm.PackageChart(cfg, out)
			if err != nil {
				return fmt.Errorf("helm package: %w", err)
			}

			fmt.Fprintf(out, "Successfully packaged chart and saved it to: %s\n", dest)

			err = writeActionOutputs(actionOutDir, chart.Name(), cfg.Version, cfg.Destination)
			if err != nil {
				return fmt.Errorf("write action output: %w", err)
			}

			return nil
		},
	}

	f := cmd.Flags()

	// Global Helm settings flags
	f.BoolVar(&cfg.Settings.Debug, "debug", false, "enables debug logs")

	// Helm dependency build flags
	f.BoolVar(&cfg.Verify, "verify", false, "verify the packages against signatures")

	// Helm package flags
	f.BoolVar(&cfg.Sign, "sign", false, "use a PGP private key to sign this package")
	f.StringVar(&cfg.SignKey, "key", "", "name of the key to use when signing. Used if --sign is true")
	f.StringVar(&cfg.Keyring, "keyring", cfg.Keyring, "location of a public keyring")
	f.StringVar(&cfg.PassphraseFile, "passphrase-file", "", `location of a file which contains the passphrase for the signing key. Use "-" in order to read from stdin.`)
	f.StringVar(&cfg.Version, "version", "", "set the version on the chart to this semver version")
	f.StringVar(&cfg.AppVersion, "app-version", "", "set the appVersion on the chart to this version")
	f.StringVar(&cfg.Destination, "destination", "", "location to write the chart")

	// Action-specific flags
	f.StringVar(&cfg.EmbedValues, "embed-values", "", "YAML object with custom chart values that should take precedence over the ones within the values.yaml")
	f.StringVar(&cfg.RegistryConfig, "registry-config", "", "Path to the CloudBees OCI registry configuration file")

	err := applyEnvVarsToFlags(cmd, envVarPrefix)
	if err != nil {
		return err
	}

	return cmd.Execute()
}

func writeActionOutputs(outputDir, chartName, chartVersion, destDir string) error {
	err := os.WriteFile(filepath.Join(outputDir, "name"), []byte(chartName), 0o640)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(outputDir, "version"), []byte(chartVersion), 0o640)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(outputDir, "directory"), []byte(destDir), 0o640)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s-%s.tgz", chartName, chartVersion)
	filePath := filepath.Join(destDir, fileName)
	err = os.WriteFile(filepath.Join(outputDir, "chart"), []byte(filePath), 0o640)
	if err != nil {
		return err
	}

	return nil
}

func applyEnvVarsToFlags(cmd *cobra.Command, envVarPrefix string) (err error) {
	nameRegex := regexp.MustCompile("[^a-zA-Z0-6]+")
	supportedEnvVars := map[string]struct{}{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		envVarName := envVarPrefix + strings.ToUpper(nameRegex.ReplaceAllString(f.Name, "_"))
		f.Usage = fmt.Sprintf("%s (%s)", f.Usage, envVarName)
		supportedEnvVars[envVarName] = struct{}{}
		if envVarValue := os.Getenv(envVarName); envVarValue != "" {
			f.DefValue = envVarValue
			e := f.Value.Set(envVarValue)
			if e != nil && err == nil {
				err = fmt.Errorf("invalid environment variable %s value provided: %w", envVarName, e)
			}
		}
	})
	if err != nil {
		_ = cmd.Usage()
		return err
	}
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, envVarPrefix) {
			kv := strings.SplitN(entry, "=", 2)
			if _, ok := supportedEnvVars[kv[0]]; !ok {
				_ = cmd.Usage()
				return fmt.Errorf("unsupported environment variable provided: %s", kv[0])
			}
		}
	}
	return nil
}
