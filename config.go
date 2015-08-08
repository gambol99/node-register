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
)

var config struct {
	// the kube api version
	kube_version string
	// a file container a token
	kube_token_file string
	// a token to use with the api
	kube_token string
	// a kube cert file
	kube_cert string
	// the kube endpoint
	kube_api string
	// insecure connection?
	kube_insecure bool
	// the port kubelet is serving health checks on
	kube_health_port int
	// enable the node reaper
	kube_node_repear bool
	// the time for a node to be offline and reaper
	kube_node_downtime time.Duration
	// the metadata used to filter the nodes
	metadata string
	// the socket for fleet
	fleet_socket string
	// the interval to wait
	time_interval time.Duration
	// the tag name
	tag_name string
	// the tag value
	tag_value string
	// show version
	show_version bool
}

const (
	DEFAULT_SYNC_INTERVAL   = time.Duration(60) * time.Second
	DEFAULT_REAPER_INTERVAL = time.Duration(1) * time.Hour
)

var (
	metadata_regex = regexp.MustCompile("^([[:alnum:]]*)=([[:alnum:]]*)$")
)

func init() {
	flag.StringVar(&config.kube_api, "api", "https://127.0.0.1:6443", "the kubernetes api endpoint to register against")
	flag.StringVar(&config.kube_token, "token", "", "a kubernetes api token to used when connecting to the endpoint")
	flag.StringVar(&config.kube_token_file, "token-file", "", "a file container a token to authenticate to kubernetes")
	flag.StringVar(&config.kube_cert, "cert", "", "a client cerfiticate to use to authenticate with kubernetes")
	flag.StringVar(&config.metadata, "metadata", "role=kubernetes", "the fleet metadata with are using to filter nodes")
	flag.StringVar(&config.fleet_socket, "fleet", "unix://var/run/fleet.sock", "the path to the fleet unix socket")
	flag.StringVar(&config.kube_version, "api-version", "v1", "the kubernetes api version")
	flag.BoolVar(&config.kube_node_repear, "node-reaper", false, "enable the removal of dead nodes from the kubernetes")
	flag.DurationVar(&config.kube_node_downtime, "reap-interval", DEFAULT_REAPER_INTERVAL, "the amount of time a node can be down before removal")
	flag.DurationVar(&config.time_interval, "interval", DEFAULT_SYNC_INTERVAL, "the amount of time in seconds to check if nodes registered")
	flag.IntVar(&config.kube_health_port, "port", 10255, "the port the kubelet is running the health endpoint on")
	flag.BoolVar(&config.show_version, "version", false, "display the node register version")
}

func parseConfig() error {
	flag.Parse()
	if config.show_version {
		fmt.Printf("node-register: %s (%s), version: %s, git+sha: %s\n", AUTHOR, EMAIL, VERSION, GIT_SHA)
		os.Exit(0)
	}

	// check: ensure the interval is great than 10 seconds
	if config.time_interval < 10 {
		return fmt.Errorf("the sync interval should be greater then 10 seconds")
	}
	// check: ensure the metadata is valid
	if matched := metadata_regex.MatchString(config.metadata); !matched {
		return fmt.Errorf("invalid metadata, should be tag=value format")
	}
	// check: ensure the token file exists
	if config.kube_token_file != "" {
		if _, err := os.Stat(config.kube_token_file); os.IsNotExist(err) {
			return fmt.Errorf("the token file: %s does not exist", config.kube_token_file)
		}
	}
	// check: ensure the cert exists
	if config.kube_cert != "" {
		if _, err := os.Stat(config.kube_cert); os.IsNotExist(err) {
			return fmt.Errorf("the kube cert file: %s does not exist", config.kube_cert)
		}
	}

	// check: ensure the url is valid
	if _, err := url.Parse(config.kube_api); err != nil {
		return fmt.Errorf("invalid url for kubernete api, error: %s", err)
	}

	// step: extract the tags
	matches := metadata_regex.FindAllStringSubmatch(config.metadata, 1)[0]
	config.tag_name = string(matches[1])
	config.tag_value = string(matches[2])

	return nil
}


