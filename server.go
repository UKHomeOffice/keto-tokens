/*
Copyright 2017 The Keto Authors

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
	"strings"
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"
	"github.com/UKHomeOffice/keto-tokens/pkg/server"

	"github.com/urfave/cli"
)

// newServiceCommand creates and returns a server command
func newServiceCommand() cli.Command {
	return cli.Command{
		Name:  "server",
		Usage: "starts the service, generating the registration tokens for kubelets",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "master",
				Usage:  "url for the kubernetes api",
				EnvVar: "KUBE_SERVICE_URL",
			},
			cli.StringFlag{
				Name:   "kube-token",
				Usage:  "kubernetes token used to authenticate to the API",
				EnvVar: "KUBE_SERVICE_TOKEN",
			},
			cli.StringFlag{
				Name:   "kubeconfig",
				Usage:  "path to a kubernetes kubeconfig for API access",
				EnvVar: "KUBECONFIG",
			},
			cli.StringFlag{
				Name:   "tag-name",
				Usage:  "resource tag used to pass kubelet token",
				Value:  "KubeletToken",
				EnvVar: "TAG_NAME",
			},
			cli.StringSliceFlag{
				Name:   "filter",
				Usage:  "collection of filter tags used to identity compute nodes (key=value)",
				EnvVar: "NODE_FILTER",
			},
			cli.StringFlag{
				Name:   "token-namespace",
				Usage:  "namespace which the registration tokens reside `NAMESPACE`",
				Value:  "kube-system",
				EnvVar: "TOKEN_NAMESPACE",
			},
			cli.DurationFlag{
				Name:   "token-ttl",
				Usage:  "the time-to-live on generate registration token",
				EnvVar: "TOKEN_TTL",
				Value:  time.Duration(30) * time.Minute,
			},
			cli.DurationFlag{
				Name:   "interval",
				Usage:  "reconcilation interval to check for compute nodes",
				Value:  time.Duration(10) * time.Second,
				EnvVar: "INTERVAL",
			},
		},
		Action: func(cx *cli.Context) error {
			return handleCommand(cx, runServiceCommand)
		},
	}
}

// runServiceCommand is the entrypoint for starting in server mode
func runServiceCommand(cx *cli.Context) error {
	p := handleCloudProvider(cx)
	// step: parse the filter tags
	tags, err := convertToTags(cx.StringSlice("filter"))
	if err != nil {
		return err
	}
	// step: create a tokens provider
	tp, err := server.NewTokenProvider()
	if err != nil {
		return err
	}

	cfg := server.Config{
		Filters:           tags,
		KubeConfig:        cx.String("kubeconfig"),
		KubeToken:         cx.String("kube-token"),
		MasterAPI:         cx.String("master"),
		ReconcileInterval: cx.Duration("interval"),
		TagName:           cx.String("tag-name"),
		TokenNamespace:    cx.String("token-namespace"),
		TokenTTL:          cx.Duration("token-ttl"),
	}

	c, err := server.New(cfg, p, tp)
	if err != nil {
		return err
	}

	return c.Start()
}

// convertToTags converts a collection of key=value to tags
func convertToTags(filters []string) (cloud.NodeTags, error) {
	tags := make(cloud.NodeTags, 0)
	for _, x := range filters {
		e := strings.Split(x, "=")
		if len(e) != 2 {
			return tags, fmt.Errorf("filter: %s is invalid", x)
		}
		if len(e[0]) == 0 || len(e[1]) == 0 {
			return tags, fmt.Errorf("filter: %s is invalid length", x)
		}
		tags[e[0]] = e[1]
	}

	return tags, nil
}
