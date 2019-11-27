// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/nodepool-labels-operator/internal/platform/errorhandler"
	"github.com/banzaicloud/nodepool-labels-operator/internal/platform/healthcheck"
	"github.com/banzaicloud/nodepool-labels-operator/internal/platform/log"
	"github.com/banzaicloud/nodepool-labels-operator/pkg/controller"
	"github.com/banzaicloud/nodepool-labels-operator/pkg/labeler"
	"github.com/banzaicloud/nodepool-labels-operator/pkg/utils"
)

// nolint: gochecknoinits
func init() {
	pflag.Bool("version", false, "Show version information")
	pflag.Bool("dump-config", false, "Dump configuration to the console")
}

func main() {
	// Loads and validates configuration
	configure()

	// Show version if asked for
	if viper.GetBool("version") {
		fmt.Printf("%s version %s (%s) built on %s\n", FriendlyServiceName, version, commitHash, buildDate)
		os.Exit(0)
	}

	// Dump config if asked for
	if viper.GetBool("dump-config") {
		c := viper.AllSettings()
		y, err := yaml.Marshal(c)
		if err != nil {
			panic(errors.WrapIf(err, "failed to dump configuration"))
		}
		fmt.Print(string(y))
		os.Exit(0)
	}

	// Create logger
	logger := log.NewLogger(configuration.Log)

	// Create error handler
	errorHandler := errorhandler.ErrorHandler(logger)
	defer emperror.HandleRecover(errorHandler)

	logger.Infof("Starting %s", FriendlyServiceName)

	// Starts health check HTTP server
	go func() {
		healthcheck.New(configuration.Healthcheck, logger, errorHandler)
	}()

	k8sconfig, err := utils.GetK8sConfig()
	emperror.Panic(err)

	clientset, err := kubernetes.NewForConfig(k8sconfig)
	emperror.Panic(err)

	nodeLabeler := labeler.New(labeler.Config{
		ManagedLabelsAnnotation: configuration.Labeler.ManagedLabelsAnnotation,
		ForbiddenLabelDomains:   configuration.Labeler.ForbiddenLabelDomains,
	}, clientset, logger, errorHandler)

	ctrl, err := controller.New(configuration.Controller, k8sconfig, nodeLabeler, logger, errorHandler)
	emperror.Panic(err)

	err = ctrl.Start()
	emperror.Panic(err)
}
