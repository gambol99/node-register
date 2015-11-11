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
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// NewKubernetesInterface creates a new client to speak to the kubernetes api service
func NewKubernetesInterface() (*KubernetesInterface, error) {
	glog.Infof("Creating a kubernetes api client, endpoint: %s", config.kubeAPI)
	// step: create a configuration for kubernetes api
	kubecfg := client.Config{
		Host:     config.kubeAPI,
		Insecure: config.kubeInsecure,
		Version:  config.kubeVersion,
	}

	// step: read in the token file is there is one
	if config.kubeTokenFile != "" {
		glog.V(4).Infof("Reading in the contents of the token file: %s", config.kubeTokenFile)
		content, err := ioutil.ReadFile(config.kubeTokenFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read the token file: %s, error: %s",
				config.kubeTokenFile, err)
		}
		glog.V(5).Infof("Using the kubernetes token from file: %s", config.kubeTokenFile)
		config.kubeToken = string(content)
	}

	// step: are we using a user token to authenticate?
	if config.kubeToken != "" {
		kubecfg.BearerToken = config.kubeToken
	}

	// step: are we using a cert to authenticate
	if config.kubeCert != "" {
		kubecfg.Insecure = false
		kubecfg.TLSClientConfig = client.TLSClientConfig{
			CAFile: config.kubeCert,
		}
	}

	// step: create the kubernetes client
	service := new(KubernetesInterface)
	kapi, err := client.New(&kubecfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create a kubernetes api client, reason: %s", err)
	}
	service.client = kapi
	return service, nil
}

// GetNodes get a list of registered kubernetes nodes
func (r KubernetesInterface) GetNodes() ([]api.Node, error) {
	nodes, err := r.client.Nodes().List(labels.Everything(), fields.Everything())
	if err != nil {
		return nil, err
	}
	return nodes.Items, err
}

// GetFailedNodes get a list of nodes in a failed state
func (r KubernetesInterface) GetFailedNodes() ([]api.Node, error) {
	// step: first get the list and then filter then
	var filtered []api.Node

	nodes, err := r.GetNodes()
	if err != nil {
		return nil, err
	}
	for _, x := range nodes {
		glog.V(6).Infof("Checking node: %s for status condition", x.Name)
		// step: skip any node without conditions
		if len(x.Status.Conditions) <= 0 {
			continue
		}
		condition := x.Status.Conditions[0]

		glog.V(6).Infof("Node condition: %V for node: %s", condition, x.Name)
		if condition.Status == api.ConditionUnknown || condition.Type != api.NodeReady {
			filtered = append(filtered, x)
		}
	}
	return filtered, nil
}

// IsRegistered checks to see if a node is registered with kubernetes
func (r KubernetesInterface) IsRegistered(name string) (*api.Node, bool, error) {
	glog.V(5).Infof("Checking if node: %s is registered with kubernetes", name)
	// step: get a list of nodes
	nodes, err := r.GetNodes()
	if err != nil {
		return nil, false, err
	}
	for _, x := range nodes {
		if x.Name == name {
			return &x, true, nil
		}
	}
	return nil, false, nil
}

// DeleteNode delete the node from kubernetes
func (r KubernetesInterface) DeleteNode(name string) error {
	glog.V(3).Infof("Deleting the node: %s from kubernetes", name)
	return r.client.Nodes().Delete(name)
}

// RegisterNode register a node with kubernetes
func (r KubernetesInterface) RegisterNode(machine *Machine) error {
	glog.V(4).Infof("Registering the machine: %s with kubernetes api", machine)

	// step: check the node is not registered already
	if _, found, err := r.IsRegistered(machine.Name); err != nil {
		return err
	} else if found {
		return fmt.Errorf("the node: %s is already registered, refusing to create it", machine.Name)
	}

	// step: construct the new kubernetes node
	node := new(api.Node)
	node.Name = machine.Name
	node.ObjectMeta.Name = machine.Name
	node.APIVersion = config.kubeVersion
	node.Labels = machine.Metadata
	node.Spec.ExternalID = machine.Name

	// step: register the node with kubernetes
	if _, err := r.client.Nodes().Create(node); err != nil {
		return err
	}

	glog.V(3).Infof("Successfully registered the node: %s with kubernetes", node.Name)
	return nil
}
