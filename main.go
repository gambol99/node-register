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
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
)

var (
	kapi *KubernetesInterface
)

func main() {
	// step: parse the configuration
	if err := parseConfig(); err != nil {
		glog.Errorf("Invalid options, %s", err)
		os.Exit(1)
	}

	glog.Infof("Starting the Node Register Service, version: %s, git+sha: %s", Version, GitSha)

	// step: create a fleet api interface
	fleet, err := NewFleetInterface()
	if err != nil {
		glog.Errorf("Failed to create fleet api, error: %s", err)
		os.Exit(1)
	}

	// step: create a client to the kubernetes api
	kapi, err = NewKubernetesInterface()
	if err != nil {
		glog.Errorf("Failed to create a kubernetes client, endpoint: %s, error: %s", config.kubeAPI, err)
		os.Exit(1)
	}

	// step: create the channel to termination requests
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		// step: are we working standalone or working for ourselve?
		if !config.standalone {
			// step: retrieve a list of machines and filter them to
			machines, err := fleet.GetMachines()
			if err != nil {
				glog.Errorf("Failed to retrieve a list of machines from fleet, error: %s", err)
				// step: jump to the next run
			}
			// step: register the machines with kubernetes
			registerMachines(machines)

		} else {
			// step: grab our machine from
			if machine, err := fleet.GetMachine(); err != nil {
				glog.Errorf("Failed to retrieve our machine from fleet error: %s", err)
			} else {
				// step: register with kubernetes
				registerMachine(machine)
			}
		}

		// step: are we reaping nodes?
		if config.kubeNodeRepear {
			// step: grab a list of nodes from kubernetes
			err := reapNodes()
			if err != nil {
				glog.Errorf("Failed to reap the nodes, error: %s", err)
			}
		}

		// wait for either a timer or a signal
		select {
		case <-signalChannel:
			glog.Infof("Recieved a shutdown signal, exiting")
			os.Exit(0)
		case <-time.After(config.timeInterval):
		}
	}
}

// reapNodes() ... remove any nodes which haven't updated for a while
func reapNodes() error {
	nodes, err := kapi.GetFailedNodes()
	if err != nil {
		return fmt.Errorf("unable to retrieve the nodes from kubernetes, error: %s", err)
	}

	glog.V(4).Infof("Found %d nodes in a failed state", len(nodes))
	for _, x := range nodes {
		condition := x.Status.Conditions[0]
		timePassed := time.Since(condition.LastHeartbeatTime.Time)
		glog.V(5).Infof("Node: %s has been down for %s", x.Name, timePassed)
		if timePassed > config.kubeNodeDowntime {
			glog.V(3).Infof("The node: %s has been down for %s, removing the node now", x.Name, timePassed)
			if err := kapi.DeleteNode(x.Name); err != nil {
				glog.Errorf("unable to remove the node: %s from kubernetes, error: %s", x.Name, err)
			}
		}
	}

	return nil
}

// registerMachines ... a wrapper for multiple machine registrations
func registerMachines(machines []*Machine) error {
	for _, machine := range machines {
		if err := registerMachine(machine); err != nil {
			glog.Errorf("Failed to register machine: %s, error: %s", machine.Name, err)
		}
	}

	return nil
}

// registerMachine() ... register the machine with Kubernetes.
//  a) the machine must match the tag filter on the metadata
//  b) we only register only if the node is responding as healthy
// 	c) if the node is already registered, we will ONLY register is the node is matched as NodeNotReady (this aides with auto scaling groups)
func registerMachine(machine *Machine) error {
	var err error

	registeredName := machine.Name

	// step: are we using dns hostname
	if config.dnsResolve {
		hostNames, err := net.LookupAddr(machine.Name)
		if err != nil {
			glog.Errorf("failed to resolve the ip address: %s, error: %s", machine.Name, err)
			return err
		}
		registeredName = hostNames[0]
	}

	// step: does the tag exist in the metadata
	if _, found := machine.Metadata[config.tagName]; !found {
		glog.V(5).Infof("Skippng machine: '%s', does not have '%s' tag in metadata", machine.Name, config.tagName)
		return nil
	}
	// step: is the value of the tag correct?
	if tag, _ := machine.Metadata[config.tagName]; tag != config.tagValue {
		glog.V(5).Infof("Skipping machine: %s, machine tag value: '%s' not equal to '%s'", machine.Name, tag, config.tagValue)
		return nil
	}

	// step: check to see if the node is healthy
	if health := nodeHealthy(machine.Name); !health {
		return fmt.Errorf("the machine: %s is marked as unhealthy, skipping the node for now", machine.Name)
	}

	// step: update the name if required
	machine.Name = registeredName

	// step: add in the environment variables before registration
	for name, value := range config.labels {
		machine.Metadata[name] = value
	}

	// step: check if the node is registered
	node, registered, err := kapi.IsRegistered(machine.Name)
	if err != nil {
		return fmt.Errorf("Unable to check if machine: %s is registered in kubernetes, error: %s", machine.Name, err)
	}

	// step: is the node already registered?
	if registered {
		nodeStatus := node.Status.Conditions[0].Type
		glog.V(4).Infof("Node: %s already register, status: %s", node.Name, nodeStatus)

		// step: the node is already registered with kubernetes - the default behaviour is to
		// check if the node status is running;
		if nodeStatus == "Ready" {
			glog.V(4).Infof("Node: %s is in a running state, refusing to register a node in a running state", node.Name)
			return nil
		}

		glog.V(4).Infof("Deleting the node: %s and registering it later", node.Name)
		// step: we delete and update node
		if err := kapi.DeleteNode(machine.Name); err != nil {
			return fmt.Errorf("Failed to delete the node: %s from kubernetes, error: %s", machine.Name, err)
		}
	}

	// step: register the node in kubernetes
	if err := kapi.RegisterNode(machine); err != nil {
		return fmt.Errorf("Failed to register the node, error: %s", err)
	}

	return nil
}

// nodeHealthy checks to see if the node in a healthy condition
func nodeHealthy(hostname string) bool {
	glog.V(4).Infof("Checking if the node: %s is in a healthy condition on port: %d", hostname, config.kubeHealthPort)
	// step: call the /healthz url
	url := fmt.Sprintf("http://%s:%d/healthz", hostname, config.kubeHealthPort)
	response, err := http.Get(url)
	if err != nil {
		glog.Errorf("Unable to check the health of the node: %s, error: %s", hostname, err)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest {
		glog.V(4).Infof("Machine: %s is healthy and responding to /healthz", hostname)
		return true
	}

	glog.V(4).Infof("Machine: %s is not in a healthy condition, response: %s", hostname, response.Body)

	return false
}
