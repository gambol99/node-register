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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
)

func main() {
	// step: parse the configuration
	if err := parseConfig(); err != nil {
		glog.Errorf("Invalid options, %s", err)
		os.Exit(1)
	}

	glog.Infof("Starting the Node Register Service, version: %s", VERSION)

	// step: create a fleet api interface
	fleet, err := newFleetInterface()
	if err != nil {
		glog.Errorf("Failed to create fleet api, error: %s", err)
		os.Exit(1)
	}

	// step: create a client to the kubernetes api
	kapi, err := newKubernetesInterface()
	if err != nil {
		glog.Errorf("Failed to create a kubernetes client, endpoint: %s, error: %s",
			config.kube_api, err)
		os.Exit(1)
	}

	// step: create the channel to termination requests
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		// step: retrieve a list of machines and filter them to
		machines, err := fleet.GetMachines()
		if err != nil {
			glog.Errorf("Failed to retrieve a list of machines from fleet, error: %s", err)
		} else {
			// step: filter out the machines and find any machines which match
			for _, machine := range machines {
				glog.V(5).Infof("Checking the machine: %s against the tags: %s=%s", machine, config.tag_name, config.tag_value)
				// step: does the tag exist in the metadata
				if _, found := machine.Metadata[config.tag_name]; !found {
					glog.V(5).Infof("Skippng machine: '%s', does not have '%s' tag in metadata", machine.Name, config.tag_name)
					continue
				}
				// step: is the value of the tag correct?
				if tag, _ := machine.Metadata[config.tag_name]; tag != config.tag_value {
					glog.V(5).Infof("Skipping machine: %s, machine tag value: '%s' not equal to '%s'", machine.Name,
						tag, config.tag_value)
					continue
				}

				// step: check to see if the node is healthy
				if health := nodeHealthy(machine); !health {
					glog.Errorf("The machine: %s is marked as unhealthy, skipping the node for now", machine.Name)
					continue
				}

				// step: check if the node is registered
				node, registered, err := kapi.IsRegistered(machine.Name)
				if err != nil {
					glog.Errorf("Unable to check if machine: %s is registered in kubernetes, error: %s", machine.Name, err)
					continue
				}

				// step: is the node already registered?
				if registered {
					node_status := node.Status.Conditions[0].Type
					glog.V(4).Infof("Node: %s already register, status: %s", node.Name, node_status)

					// step: the node is already registered with kubernetes - the default behaviour is to
					// check if the node status is running;
					if node_status == "Ready" {
						glog.V(4).Infof("Node: %s is in a running state, refusing to register a node in a running state", node.Name)
						continue
					}
					glog.V(4).Infof("Deleting the node: %s and registering it later", node.Name)
					// step: we delete and update node
					if err := kapi.DeleteNode(machine.Name); err != nil {
						glog.Errorf("Failed to delete the node: %s from kubernetes, error: %s", machine.Name, err)
						continue
					}
				}
				// step: register the node in kubernetes
				if err := kapi.RegisterNode(machine); err != nil {
					glog.Errorf("Failed to register the node, error: %s", err)
				}
			}
		}
		// wait for either a timer or a signal
		select {
		case <-signalChannel:
			glog.Infof("Recieved a shutdown signal, exiting")
			os.Exit(0)
		case <- time.After(time.Duration(config.time_interval) * time.Second):
		}
	}
}

// nodeHealthy() ... checks to see if the node in a healthy condition
func nodeHealthy(machine *Machine) bool {
	glog.V(4).Infof("Checking if the node: %s is in a healthy condition on port: %d", machine.Name, config.kube_health_port)
	// step: call the /healthz url
	url := fmt.Sprintf("http://%s:%d/healthz", machine.Name, config.kube_health_port)
	response, err := http.Get(url)
	if err != nil {
		glog.Errorf("Unable to check the health of the node: %s, error: %s", machine.Name, err)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest {
		glog.V(4).Infof("Machine: %s is healthy and responding to /healthz", machine.Name)
		return true
	}

	glog.V(4).Infof("Machine: %s is not in a healthy condition, response: %s", machine.Name, response.Body)
	return false
}
