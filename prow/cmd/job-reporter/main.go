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
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/pkg/flagutil"
	prowjobinformer "k8s.io/test-infra/prow/client/informers/externalversions"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/crier"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/logrusutil"
)

type options struct {
	configPath    string
	jobConfigPath string
	skipReport    bool

	dryRun     bool
	kubernetes prowflagutil.KubernetesOptions
	github     prowflagutil.GitHubOptions // TODO(fejta): remove
	gitee      prowflagutil.GiteeOptions

	githubWorkers int
	giteeWorkers  int
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options

	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml.")
	fs.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to prow job configs.")
	fs.BoolVar(&o.skipReport, "skip-report", false, "Validate that crier is reporting to github, not plank")

	fs.IntVar(&o.githubWorkers, "github-workers", 0, "Number of github report workers (0 means disabled)")
	fs.IntVar(&o.giteeWorkers, "gitee-workers", 0, "Number of gitee report workers (0 means disabled)")

	fs.BoolVar(&o.dryRun, "dry-run", true, "Whether or not to make mutating API calls to GitHub.")
	for _, group := range []flagutil.OptionGroup{&o.kubernetes, &o.github, &o.gitee} {
		group.AddFlags(fs)
	}

	fs.Parse(args)
	return o
}

func (o *options) Validate() error {
	o.github.AllowAnonymous = true
	for _, group := range []flagutil.OptionGroup{&o.kubernetes, &o.github, &o.gitee} {
		if err := group.Validate(o.dryRun); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	logrusutil.ComponentInit()

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	if o.skipReport {
		return
	}

	var configAgent config.Agent
	if err := configAgent.Start(o.configPath, o.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}

	err := run(&o, configAgent.Config)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating reporter")
	}
}

func run(o *options, cfg config.Getter) error {
	reporters, err := buildReporter(o, cfg)
	if err != nil {
		return err
	}

	if len(reporters) == 0 {
		return fmt.Errorf("should have at least one job reporter to run.")
	}

	prowjobClientset, err := o.kubernetes.ProwJobClientset(cfg().ProwJobNamespace, o.dryRun)
	if err != nil {
		return fmt.Errorf("prow client: %w", err)
	}

	defer interrupts.WaitForGracefulShutdown()

	const resync = 0 * time.Minute
	prowjobInformerFactory := prowjobinformer.NewSharedInformerFactoryWithOptions(
		prowjobClientset, resync, prowjobinformer.WithNamespace(cfg().ProwJobNamespace))

	prowjobInformerFactory.Start(interrupts.Context().Done())

	for r, n := range reporters {
		c := crier.NewController(
			prowjobClientset,
			kube.RateLimiter(r.GetName()),
			prowjobInformerFactory.Prow().V1().ProwJobs(),
			r, n)

		interrupts.Run(func(ctx context.Context) { c.Run(ctx) })
	}
	return nil
}
