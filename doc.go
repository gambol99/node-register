/*
Copyright 2014 Rohith All rights reserved.

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

import (
	"net/http"

	fleet "github.com/coreos/fleet/client"
	kube "k8s.io/kubernetes/pkg/client"
)

// FleetInterface ... is the interface used to extract the machines from fleet cluster
type FleetInterface struct {
	// the http client
	httpClient *http.Client
	// the fleet client
	fleetClient fleet.API
}

// KubernetesInterface ... the interface to speak to the kubernetes api
type KubernetesInterface struct {
	// the kubernetes api
	client *kube.Client
}

// Machine ... the structure of a machine from fleet
type Machine struct {
	// the name of the machine - the ip address
	Name string
	// the metadata associated to the machine
	Metadata map[string]string
}
