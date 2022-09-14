/*
Copyright (c) 2022 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package secret

import (
	"github.com/nordix/meridio/pkg/k8s/watcher"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CreateSecretInterface -
// Creates a v1.SecretInterface in order to build a k8s object monitor for Secrets
func CreateSecretInterface(namespace string) (watcher.WatchObject, error) {
	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		return nil, err
	}

	return clientset.CoreV1().Secrets(namespace), err

}
