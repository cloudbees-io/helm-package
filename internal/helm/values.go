package helm

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
)

const (
	inputValuesURLScheme = "actioninput"
	inputValuesURL       = inputValuesURLScheme + ":values"
)

func mergeChartValues(valuesFile, customValuesYAML string, getters getter.Providers) (map[string]interface{}, error) {
	var customValues map[string]interface{}
	err := yaml.Unmarshal([]byte(customValuesYAML), &customValues)
	if err != nil {
		return nil, fmt.Errorf("unmarshal values from action input: %w", err)
	}

	valueGetters := append(getters, getter.Provider{
		Schemes: []string{inputValuesURLScheme},
		New: func(_ ...getter.Option) (getter.Getter, error) {
			return embeddedValuesGetter(customValues), nil
		},
	})
	valueOpts := &values.Options{
		ValueFiles: []string{
			valuesFile,
			inputValuesURL,
		},
	}
	vals, err := valueOpts.MergeValues(valueGetters)
	if err != nil {
		return nil, errors.Wrap(err, "load values")
	}
	return vals, nil
}

type embeddedValuesGetter map[string]interface{}

func (g embeddedValuesGetter) Get(url string, options ...getter.Option) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	if url == inputValuesURL {
		if g != nil {
			b, err := yaml.Marshal(map[string]interface{}(g))
			if err != nil {
				return buf, errors.Wrap(err, "marshal inline helm values")
			}
			_, err = buf.Write(b)
			if err != nil {
				return nil, err
			}
		}
		return buf, nil
	}
	return buf, errors.Errorf("unsupported URL %q provided to actioninput values getter", url)
}
