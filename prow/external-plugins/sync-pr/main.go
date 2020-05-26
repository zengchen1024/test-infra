/*
Copyright 2017 The Kubernetes Authors.

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
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
)

type options struct {
	port   int
	dryRun bool

	github prowflagutil.GitHubOptions
	gitee  prowflagutil.GiteeOptions

	syncprConfig      string
	webhookSecretFile string
}

func (o *options) Validate() error {
	for _, group := range []flagutil.OptionGroup{&o.github, &o.gitee} {
		if err := group.Validate(o.dryRun); err != nil {
			return err
		}
	}

	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.port, "port", 8888, "Port to listen on.")
	fs.StringVar(&o.syncprConfig, "syncpr-config", "/etc/plugins/plugins.yaml", "Path to plugin config file.")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.webhookSecretFile, "hmac-secret-file", "/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")
	for _, group := range []flagutil.OptionGroup{&o.github, &o.gitee} {
		group.AddFlags(fs)
	}
	fs.Parse(os.Args[1:])
	return o
}

func main() {
	o := gatherOptions()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("plugin", "syncpr")

	config, err := load(o.syncprConfig)
	if err != nil {
		logrus.WithError(err).Fatal("Error loading syncpr config.")
	}

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.github.TokenPath, o.gitee.TokenPath, o.webhookSecretFile}); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}
	gitClient, err := o.github.GitClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Git client.")
	}

	giteeClient, err := o.gitee.GiteeClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Gitee client.")
	}
	giteeGitClient, err := o.gitee.GitClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Gitee Git client.")
	}

	name, err := giteeClient.BotName()
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Gitee botname.")
	}

	serv := &server{
		tokenGenerator: secretAgent.GetTokenGenerator(o.webhookSecretFile),
		config:         func() syncPRConfig { return config },
		ghc:            githubClient,
		ghgc:           git.ClientFactoryFrom(gitClient),
		gec:            giteeClient,
		gegc:           giteeGitClient,
		log:            log,
		robot:          name,
	}

	mux := http.NewServeMux()
	mux.Handle("/", serv)
	externalplugins.ServeExternalPluginHelp(mux, log, helpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}
	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}
