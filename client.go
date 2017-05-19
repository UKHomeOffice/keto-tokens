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
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/client"

	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

// newClientCommand create's and returns the client sub-command
func newClientCommand() cli.Command {
	return cli.Command{
		Usage: "retrieves a kubenetes registration tokens for compute kubelets",
		Name:  "client",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "master",
				Usage:  "url for the kubernetes API `URL`",
				Value:  "https://127.0.0.1:6443",
				EnvVar: "KUBE_SERVICE_URL",
			},
			cli.StringFlag{
				Name:   "kubeconfig",
				Usage:  "path to write out the kubeconfig `PATH`",
				Value:  "kubeconfig-bootstrap",
				EnvVar: "KUBECONFIG",
			},
			cli.StringFlag{
				Name:   "tag-name",
				Usage:  "tag used to pass the kubelet registration `NAME`",
				Value:  "KubeletToken",
				EnvVar: "TAG_NAME",
			},
			cli.StringFlag{
				Name:   "ca-path",
				Usage:  "path to file containing kubeapi ca certificate (otherwise skip-tls-verify is used)",
				EnvVar: "CA_PATH",
			},
			cli.DurationFlag{
				Name:   "interval",
				Usage:  "interval for checking for resource tags `DURATION`",
				Value:  time.Duration(5) * time.Second,
				EnvVar: "INTERVAL",
			},
			cli.DurationFlag{
				Name:   "timeout",
				Usage:  "optional timeout for the operation `DURATION`",
				EnvVar: "TIMEOUT",
			},
		},
		Action: func(cx *cli.Context) error {
			return handleCommand(cx, runClientAction)
		},
	}
}

// runClientAction performs the client mode operation
func runClientAction(cx *cli.Context) error {
	p := handleCloudProvider(cx)
	// step: create a new client
	cfg := client.Config{
		Interval: cx.Duration("interval"),
		TagName:  cx.String("tag-name"),
		Timeout:  cx.Duration("timeout"),
	}
	c, err := client.New(cfg, p)
	if err != nil {
		return err
	}

	// step: attempt to consume the client token
	log.Infof("attempting to get registration token, timeout: %s, tag: %s", cfg.Timeout, cfg.TagName)
	token, err := c.Start()
	if err != nil {
		if err != client.ErrConsumedToken {
			return err
		}
		log.Warn("kubelet registration token already consumed, skipping kubeconfig")
		return nil
	}

	// step: get the inputs
	masterURL := cx.String("master")
	caPath := cx.String("ca-path")
	kubeConfig := cx.String("kubeconfig")

	// step: ensure any directory structure for kube config
	if err = os.MkdirAll(path.Dir(kubeConfig), os.FileMode(0775)); err != nil {
		return err
	}

	// step: are we writing out a kube config?
	log.Infof("retrieved registration token, writing kubeconfig: %s", kubeConfig)
	content, err := client.GenerateKubeconfig(token, masterURL, caPath)
	if err == nil {
		if err = ioutil.WriteFile(kubeConfig, content, os.FileMode(0640)); err != nil {
			return err
		}
	}

	return nil
}
