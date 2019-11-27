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

package labeler

import (
	"encoding/json"
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/nodepool-labels-operator/internal/platform/log"
)

const (
	managedLabelsAnnotation = "nodepool.banzaicloud.io/managed-labels"
)

// Labeler describes the node labeler
type Labeler struct {
	managedLabelsAnnotation string
	forbiddenLabelDomains   []string

	clientset    kubernetes.Interface
	logger       log.Logger
	errorHandler emperror.Handler
}

// New gives back an initialized Labeler
func New(config Config, clientset kubernetes.Interface, logger log.Logger, errorHandler emperror.Handler) *Labeler {
	annotation := config.ManagedLabelsAnnotation
	if annotation == "" {
		annotation = managedLabelsAnnotation
	}

	return &Labeler{
		managedLabelsAnnotation: annotation,
		forbiddenLabelDomains:   config.ForbiddenLabelDomains,

		clientset:    clientset,
		logger:       logger,
		errorHandler: errorHandler,
	}
}

// SyncLabels syncs node labels
func (l *Labeler) SyncLabels(node *api_v1.Node, labelsToSet map[string]string) error {
	l.logger.WithField("node", node.Name).Debug("sync labels")

	oldData, err := json.Marshal(*node)
	if err != nil {
		return errors.WrapIf(err, "could not marshal old node object")
	}

	nodeLabels, managedLabels := l.getDesiredLabels(node, labelsToSet)
	annotations, err := l.updateAnnotations(node.GetAnnotations(), managedLabels)
	if err != nil {
		return errors.WrapIf(err, "could not update annotations")
	}
	node.SetAnnotations(annotations)
	node.SetLabels(nodeLabels)

	newData, err := json.Marshal(*node)
	if err != nil {
		return errors.WrapIf(err, "could not marshal new node object")
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, *node)
	if err != nil {
		return errors.WrapIf(err, "could not create two way merge patch")
	}

	_, err = l.clientset.CoreV1().Nodes().Patch(node.Name, types.MergePatchType, patch)
	if err != nil {
		return errors.WrapIf(err, "could not patch node")
	}

	return nil
}

func (l *Labeler) updateAnnotations(currentAnnotations map[string]string, managedLabels []string) (map[string]string, error) {
	if currentAnnotations == nil {
		currentAnnotations = make(map[string]string)
	}

	managedLabelsJSON, err := json.Marshal(managedLabels)
	if err != nil {
		return currentAnnotations, errors.WrapIf(err, "could not marshal managed labels to annotation")
	}
	currentAnnotations[l.managedLabelsAnnotation] = string(managedLabelsJSON)

	return currentAnnotations, nil
}

func (l *Labeler) getDesiredLabels(node *api_v1.Node, labelsToSet map[string]string) (map[string]string, []string) {
	logger := l.logger.WithField("node", node.Name)
	managedLabels, _ := l.getManagedLabels(node)
	nodeLabels := node.GetLabels()
	mLabels := make(map[string]bool, len(managedLabels))
	for _, label := range managedLabels {
		mLabels[label] = true
	}

	for label := range nodeLabels {
		if mLabels[label] && len(labelsToSet[label]) == 0 {
			logger.WithField("label", label).Info("removing label")
			delete(nodeLabels, label)
		}
	}

	managedLabels = make([]string, 0)
	for label, value := range labelsToSet {
		logger = logger.WithFields(log.Fields{
			"label":      label,
			"labelValue": value,
		})
		if !l.isLabelAllowed(label) {
			logger.Info("forbidden label")
			continue
		}
		managedLabels = append(managedLabels, label)
		if nodeLabels[label] == value {
			continue
		}
		logger.Info("setting label")
		nodeLabels[label] = value
	}

	return nodeLabels, managedLabels
}

func (l *Labeler) getManagedLabels(node *api_v1.Node) ([]string, error) {
	var labels []string

	err := json.Unmarshal([]byte(node.GetAnnotations()[l.managedLabelsAnnotation]), &labels)
	if err != nil {
		return labels, errors.WrapIf(err, "could not unmarshal annotation")
	}

	return labels, nil
}

func (l *Labeler) isLabelAllowed(label string) bool {
	for _, domain := range l.forbiddenLabelDomains {
		if strings.Contains(label, domain) {
			return false
		}
	}

	return true
}
