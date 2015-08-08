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
	"net/url"
	"time"

	fleet "github.com/coreos/fleet/client"
	"github.com/golang/glog"
)

// newFleetInterface ... creates a new interface to interact to the fleet cluster service
func newFleetInterface() (*FleetInterface, error) {
	glog.V(3).Infof("Creating a client to fleet service, endpoint: %s", config.fleetSocket)
	service := new(FleetInterface)

	// step: parse the verify the fleet endpoint
	location, err := url.Parse(config.fleetSocket)
	if err != nil {
		return nil, err
	}

	// step: ensure we are using a fleet socket
	if location.Scheme != "unix" {
		return nil, fmt.Errorf("the fleet endpoint should be a unix socket file, please read documentation")
	}

	location.Scheme = "http"
	location.Host = "domain-sock"
	socket_path := location.Path
	location.Path = ""

	// step: create the http client
	service.http_client = &http.Client{
		Timeout: time.Duration(10) * time.Second,
		Transport: &http.Transport{
			Dial:	func(network, addr string) (net.Conn, error) {
				return net.Dial("unix", socket_path)
			},
			DisableKeepAlives: true,
		},
	}

	// step: create the fleet client
	service.fleet_client, err = fleet.NewHTTPClient(service.http_client, *location)
	if err != nil {
		return nil, fmt.Errorf("unable to create the fleet api client, error: %s", err)
	}
	return service, nil
}

// GetMachine ... get my machine from fleet
func (r FleetInterface) GetMachine() (*Machine, error) {
	// step: get all the machines
	machines, err := r.GetMachines()
	if err != nil {
		return nil, err
	}
	// step: iterate and find the machine
	for _, machine := range machines {
		if machine.Name == config.fleetIpAddress {
			return machine, nil
		}
	}

	return nil, fmt.Errorf("unable to find the machine: %s in the list of machines", config.fleetIpAddress)
}


// GetMachines ... return a list of machines from fleet
func (r FleetInterface) GetMachines() ([]*Machine, error) {
	glog.V(5).Infof("Retrieving a list of the machines in the fleet cluster")
	// step: get the list of machines
	machines, err := r.fleet_client.Machines()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve a list of machines from fleet, error: %s", err)
	}
	// step: constructing a list of machine
	list := make([]*Machine, 0)
	for _, x := range machines {
		machine := &Machine{
			Name:     x.PublicIP,
			Metadata: x.Metadata,
		}
		glog.V(6).Infof("Adding the machine: %s to the list of fleet nodes", x)
		list = append(list, machine)
	}
	glog.V(4).Infof("Found %d machine in the fleet cluster", len(machines))
	return list, nil
}
