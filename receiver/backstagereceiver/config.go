// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package backstagereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/backstagereceiver"

import (
	"time"

	"go.einride.tech/backstage/catalog"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type mode string

const (
	PullMode  mode = "pull"
	WatchMode mode = "watch"

	defaultPullInterval    time.Duration = time.Hour
	defaultMode            mode          = PullMode
	defaultResourceVersion               = "1"
)

type BackstageAPIConfig struct {
	URL   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
}

type BackstageConfig struct {
	Kind            string        `mapstructure:"kind"`
	Group           string        `mapstructure:"group"`
	Namespaces      []string      `mapstructure:"namespaces"`
	Filters         []string      `mapstructure:"filters"`
	Fields          []string      `mapstructure:"fields"`
	Interval        time.Duration `mapstructure:"interval"`
	ResourceVersion string        `mapstructure:"resource_version"`
	// gvr             *schema.GroupVersionResource
}

type Config struct {
	BackstageAPIConfig `mapstructure:",squash"`

	Objects []*BackstageConfig `mapstructure:"objects"`

	// For mocking purposes only.
	makeClient func() (catalog.Client, error)
}

// TODO
func (c *Config) Validate() error {

	return nil

	// validObjects, err := c.getValidObjects()
	// if err != nil {
	// 	return err
	// }
	// for _, object := range c.Objects {
	// 	gvrs, ok := validObjects[object.Kind]
	// 	if !ok {
	// 		availableResource := make([]string, len(validObjects))
	// 		for k := range validObjects {
	// 			availableResource = append(availableResource, k)
	// 		}
	// 		return fmt.Errorf("resource %v not found. Valid resources are: %v", object.Kind, availableResource)
	// 	}

	// 	gvr := gvrs[0]
	// 	for i := range gvrs {
	// 		if gvrs[i].Group == object.Group {
	// 			gvr = gvrs[i]
	// 			break
	// 		}
	// 	}

	// 	if object.Interval == 0 {
	// 		object.Interval = defaultPullInterval
	// 	}

	// 	object.gvr = gvr
	// }
	// return nil
}

func (c *Config) getClient() (catalog.Client, error) {

	opts := []catalog.ClientOption{
		catalog.WithBaseURL(c.URL),
	}

	if c.Token != "" {
		opts = append(opts, catalog.WithToken(c.Token))
	}

	client := catalog.NewClient(opts...)

	return *client, nil
}

func (c *Config) getValidObjects() (map[string][]*schema.GroupVersionResource, error) {
	component := schema.GroupVersionResource{
		Group:    "backstage.io",
		Version:  "v1alpha1",
		Resource: "Component",
	}

	// TODO fix me
	hardcoded := map[string][]*schema.GroupVersionResource{
		"component.backstage.io/v1alpha1": {&component},
	}

	return hardcoded, nil
}
