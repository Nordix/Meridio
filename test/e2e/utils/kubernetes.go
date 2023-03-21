/*
Copyright (c) 2021 Nordix Foundation

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

package utils

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetClientSet() (*kubernetes.Clientset, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func PodExec(pod *corev1.Pod, container string, command []string) (string, error) {
	clientCfg, err := config.GetConfig()
	if err != nil {
		logrus.Errorf("Unable to get clientCfg: %v", err)
	}
	clientSet, err := GetClientSet()
	if err != nil {
		logrus.Errorf("Unable to get clientSet: %v", err)
	}

	req := clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")
	scheme := runtime.NewScheme()
	err = corev1.AddToScheme(scheme)
	if err != nil {
		return "", err
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: container,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(clientCfg, "POST", req.URL())
	if err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("%w; %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func PodHasNetworkInterface(pod *corev1.Pod, container string, interfaceSubName string) (bool, error) {
	interfaces, err := PodExec(pod, "example-target", []string{"ip", "-o", "link", "show"})
	if err != nil {
		return false, err
	}
	return strings.Contains(interfaces, "nsc"), nil
}
