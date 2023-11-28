// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ironcore

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type cloudProviderConfig struct {
	RestConfig  *rest.Config
	Namespace   string
	cloudConfig CloudConfig
}

type CloudConfig struct {
	NetworkName string `json:"networkName"`
	PrefixName  string `json:"prefixName,omitempty"`
	ClusterName string `json:"clusterName"`
}

var (
	IroncoreKubeconfigPath string
)

func AddExtraFlags(fs *pflag.FlagSet) {
	fs.StringVar(&IroncoreKubeconfigPath, "ironcore-kubeconfig", "", "Path to the ironcore kubeconfig.")
}

func LoadCloudProviderConfig(f io.Reader) (*cloudProviderConfig, error) {
	klog.V(2).Infof("Reading configuration for cloud provider: %s", ProviderName)
	configBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read in config")
	}

	cloudConfig := &CloudConfig{}
	if err := yaml.Unmarshal(configBytes, cloudConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cloud config: %w", err)
	}

	if cloudConfig.NetworkName == "" {
		return nil, fmt.Errorf("networkName missing in cloud config")
	}

	if cloudConfig.ClusterName == "" {
		return nil, fmt.Errorf("clusterName missing in cloud config")
	}

	ironcoreKubeconfigData, err := os.ReadFile(IroncoreKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ironcore kubeconfig %s: %w", IroncoreKubeconfigPath, err)
	}

	ironcoreKubeconfig, err := clientcmd.Load(ironcoreKubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("unable to read ironcore cluster kubeconfig: %w", err)
	}
	clientConfig := clientcmd.NewDefaultClientConfig(*ironcoreKubeconfig, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to serialize ironcore cluster kubeconfig: %w", err)
	}
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get ironcore cluster rest config: %w", err)
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace from ironcore kubeconfig: %w", err)
	}
	// TODO: empty or unset namespace will be defaulted to the 'default' namespace. We might want to handle this
	// as an error.
	if namespace == "" {
		return nil, fmt.Errorf("got a empty namespace from ironcore kubeconfig")
	}
	klog.V(2).Infof("Successfully read configuration for cloud provider: %s", ProviderName)

	return &cloudProviderConfig{
		RestConfig:  restConfig,
		Namespace:   namespace,
		cloudConfig: *cloudConfig,
	}, nil
}
