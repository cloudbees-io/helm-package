package helm

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"

	"github.com/cloudbees-io/registry-config/pkg/registries"
	"github.com/cloudbees-io/registry-config/pkg/resolve"
)

// All finds all of the registered getters as a list of Provider instances.
// Currently, the built-in getters and the discovered plugins with downloader
// notations are collected.
// As opposed to the upstream Helm implementation, the returned OCI getter honours the registry mirror configuration.
func All(settings *cli.EnvSettings, registryConfigFile string) getter.Providers {
	providers := getter.All(settings)
	replaceOCIGetter(providers, registryConfigFile)
	return providers
}

func replaceOCIGetter(providers getter.Providers, registryConfigFile string) {
	for i, p := range providers {
		if slices.Contains(p.Schemes, registry.OCIScheme) {
			providers[i] = getter.Provider{
				Schemes: []string{registry.OCIScheme},
				New:     newOCIGetterFactory(registryConfigFile),
			}
			return
		}
	}
}

func newOCIGetterFactory(registryConfigFile string) getter.Constructor {
	return func(opts ...getter.Option) (getter.Getter, error) {
		client, err := getter.NewOCIGetter(opts...)
		if err != nil {
			return nil, err
		}

		registryConfig := registries.Config{}

		if registryConfigFile != "" {
			registryConfig, err = registries.LoadConfig(registryConfigFile)
			if err != nil {
				return nil, err
			}
		}

		imageRefRewriter, err := resolve.NewResolver(registryConfig)
		if err != nil {
			return nil, err
		}

		return &ociGetter{
			delegate:    client,
			refRewriter: imageRefRewriter,
		}, nil
	}
}

type ociGetter struct {
	delegate    getter.Getter
	refRewriter *resolve.Resolver
}

// Get performs a Get from repo.Getter and returns the body.
func (g *ociGetter) Get(href string, options ...getter.Option) (*bytes.Buffer, error) {
	ref := strings.TrimPrefix(href, fmt.Sprintf("%s://", registry.OCIScheme))
	locations, err := g.refRewriter.Resolve(ref)
	if err != nil {
		return nil, err
	}

	var errs []error

	for _, location := range locations {
		uri := fmt.Sprintf("%s://%s", registry.OCIScheme, location)

		result, err := g.delegate.Get(uri, options...)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		return result, nil
	}

	return nil, fmt.Errorf("failed to resolve OCI ref: %w", errors.Join(errs...))
}
