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
	"os"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"
	_ "github.com/UKHomeOffice/keto-tokens/pkg/cloud/aws"

	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

// newKetoTokens returns a cli application
func newKetoTokens() *cli.App {
	app := cli.NewApp()
	app.Name = prog
	app.Usage = "is a client/server used to generate and consume kubelet registration tokens"
	app.Author = author
	app.Version = fmt.Sprintf("%s (git+sha: %s)", release, gitsha)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			EnvVar: "CLOUD_PROVIDER",
			Name:   "c, cloud",
			Usage:  "specify the cloud provider (aws, gce) `NAME`",
			Value:  "aws",
		},
		cli.BoolFlag{
			Name:   "verbose",
			Usage:  "switch on verbose logging mode `BOOL`",
			EnvVar: "VERBOSE",
		},
	}
	app.Before = func(cx *cli.Context) error {
		if cx.Bool("verbose") {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	}

	app.Commands = []cli.Command{
		newServiceCommand(),
		newClientCommand(),
	}

	return app
}

// handleCommand is a generic wrapper for handling commands, or more precisely their errors
func handleCommand(cx *cli.Context, cmd func(*cli.Context) error) error {
	// step: handle any panics in the command
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[error] %s", r)
			os.Exit(1)
		}
	}()
	// step: call the command and handle any errors
	if err := cmd(cx); err != nil {
		printError("operation failed, error: %s", err)
	}

	return nil
}

// handleCloudProvider retrieves a cloud provider for us
func handleCloudProvider(cx *cli.Context) cloud.Provider {
	p, err := cloud.Get(cx.GlobalString("cloud"))
	if err != nil {
		panic(err)
	}

	return p
}

func printError(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[error] "+message+"\n", args...)
	os.Exit(1)
}
