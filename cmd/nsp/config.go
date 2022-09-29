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

package main

import "time"

// Config for the NSP
type Config struct {
	Namespace     string        `default:"default" desc:"Namespace the pod is running on" split_words:"true"`
	Port          string        `default:"7778" desc:"Trench the pod is running on" split_words:"true"`
	ConfigMapName string        `default:"meridio-configuration" desc:"Name of the ConfigMap containing the configuration" split_words:"true"`
	Datasource    string        `default:"/run/nsp/data/registry.db" desc:"Path and file name of the sqlite database" split_words:"true"`
	LogLevel      string        `default:"DEBUG" desc:"Log level" split_words:"true"`
	EntryTimeout  time.Duration `default:"60s" desc:"Timeout of the entries" split_words:"true"`

	ProfilingEnabled bool `default:"false" desc:"enable profiling" split_words:"true"`
	ProfilingPort    int  `default:"9995" desc:"port of the profiling http server" split_words:"true"`
}
