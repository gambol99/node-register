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
	"flag"
	"fmt"
	"regexp"
	"os"
	"net/url"
	"time"
	"strings"
)

var config struct {
	// the kube api version
	kubeVersion string
	// a file container a token
	kubeTokenFile string
	// a token to use with the api
	kubeToken string
	// a kube cert file
	kubeCert string
	// the kube endpoint
	kubeApi string
	// insecure connection?
	kubeInsecure bool
	// the port kubelet is serving health checks on
	kubeHealthPort int
	// enable the node reaper
	kubeNodeRepear bool
	// the time for a node to be offline and reaper
	kubeNodeDowntime time.Duration
	// the metadata used to filter the nodes
	metadata string
	// the socket for fleet
	fleetSocket string
	// the interface fleet is using as public ip
	fleetInterface string
	// the public ip address of fleet
	fleetIpAddress string
	// the interval to wait
	timeInterval time.Duration
	// the tag name
	tagName string
	// the tag value
	tagValue string
	// show version
	showVersion bool
	// standalone
	standalone bool
	// environment variables
	labels map[string]string
}

const (
	DEFAULT_SYNC_INTERVAL   = time.Duration(60) * time.Second
	DEFAULT_REAPER_INTERVAL = time.Duration(1) * time.Hour
)

var (
	metadata_regex = regexp.MustCompile("^([[:alnum:]]*)=([[:alnum:]]*)$")
)

func init() {
	parseEnvironmentVars(os.Environ())
	flag.StringVar(&config.kubeApi, "api", "https://127.0.0.1:6443", "the kubernetes api endpoint to register against")
	flag.StringVar(&config.kubeToken, "token", "", "a kubernetes api token to used when connecting to the endpoint")
	flag.StringVar(&config.kubeTokenFile, "token-file", "", "a file container a token to authenticate to kubernetes")
	flag.StringVar(&config.kubeCert, "cert", "", "a client cerfiticate to use to authenticate with kubernetes")
	flag.StringVar(&config.metadata, "metadata", "role=kubernetes", "the fleet metadata with are using to filter nodes")
	flag.StringVar(&config.fleetSocket, "fleet", "unix://var/run/fleet.sock", "the path to the fleet unix socket")
	flag.StringVar(&config.fleetInterface, "interface", "", "you can either specify the interface and we'll grab the ip address or the ip below")
	flag.StringVar(&config.fleetIpAddress, "address", "", "the public ip address using by fleet, only used on standalone mode")
	flag.StringVar(&config.kubeVersion, "api-version", "v1", "the kubernetes api version")
	flag.BoolVar(&config.standalone, "standalone", false, "switch the service into standalone mode, i.e. we only register ourself")
	flag.BoolVar(&config.kubeNodeRepear, "node-reaper", false, "enable the removal of dead nodes from the kubernetes")
	flag.DurationVar(&config.kubeNodeDowntime, "reap-interval", DEFAULT_REAPER_INTERVAL, "the amount of time a node can be down before removal")
	flag.DurationVar(&config.timeInterval, "interval", DEFAULT_SYNC_INTERVAL, "the amount of time in seconds to check if nodes registered")
	flag.IntVar(&config.kubeHealthPort, "port", 10255, "the port the kubelet is running the health endpoint on")
	flag.BoolVar(&config.showVersion, "version", false, "display the node register version")
}

// parseEnvironmentVars ... looks for any environment variables prefixed with NODE_REGISTER_<NAME>=VALUE
func parseEnvironmentVars(envs []string) {
	config.labels = make(map[string]string, 0)

	var regex = regexp.MustCompile("^NODE_REGISTER_(.*)=(.*)$")
	// step: iterate the environment variables
	for _, key_name := range envs {
		// check if 'no' match and continue
		if matched := regex.MatchString(key_name); !matched {
			continue
		}

		// step: grab the matches
		matches := regex.FindAllStringSubmatch(key_name, -1)

		config.labels[strings.ToLower(matches[0][1])] = matches[0][2]
	}
}

func parseConfig() error {
	var err error

	flag.Parse()
	if config.showVersion {
		fmt.Printf("node-register: %s (%s), version: %s, git+sha: %s\n", Author, Email, Version, GitSha)
		os.Exit(0)
	}

	// check: ensure the interval is great than 10 seconds
	if config.timeInterval < 10 {
		return fmt.Errorf("the sync interval should be greater then 10 seconds")
	}
	// check: ensure the metadata is valid
	if matched := metadata_regex.MatchString(config.metadata); !matched {
		return fmt.Errorf("invalid metadata, should be tag=value format")
	}
	// check: ensure the token file exists
	if config.kubeTokenFile != "" {
		if _, err := os.Stat(config.kubeTokenFile); os.IsNotExist(err) {
			return fmt.Errorf("the token file: %s does not exist", config.kubeTokenFile)
		}
	}
	// check: ensure the cert exists
	if config.kubeCert != "" {
		if _, err := os.Stat(config.kubeCert); os.IsNotExist(err) {
			return fmt.Errorf("the kube cert file: %s does not exist", config.kubeCert)
		}
	}

	// check: ensure the url is valid
	if _, err := url.Parse(config.kubeApi); err != nil {
		return fmt.Errorf("invalid url for kubernete api, error: %s", err)
	}

	// step: extract the tags
	matches := metadata_regex.FindAllStringSubmatch(config.metadata, 1)[0]
	config.tagName = string(matches[1])
	config.tagValue = string(matches[2])

	// step: if we are running in standalone more, we need the ip address
	if config.standalone {
		// check: we need interface or ip set
		if config.fleetIpAddress == "" && config.fleetInterface == "" {
			return fmt.Errorf("you have to set either fleet interface or the public ip address")
		}

		if config.fleetInterface != "" {
			config.fleetIpAddress, err = getInterfaceAddress(config.fleetInterface)
			if err != nil {
				return err
			}
		}
	}

	return nil
}


