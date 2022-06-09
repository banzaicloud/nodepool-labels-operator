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

package controller

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	api_v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/banzaicloud/nodepool-labels-operator/internal/platform/log"
	"github.com/banzaicloud/nodepool-labels-operator/pkg/apis/nodepoollabelset/v1alpha1"
	npls_clientset "github.com/banzaicloud/nodepool-labels-operator/pkg/client/clientset/versioned"
	informers "github.com/banzaicloud/nodepool-labels-operator/pkg/client/informers/externalversions/nodepoollabelset/v1alpha1"
	"github.com/banzaicloud/nodepool-labels-operator/pkg/labeler"
)

// Controller manages node pool labels
type Controller struct {
	namespace          string
	nodepoolNameLabels []string

	k8sConfig *rest.Config
	labeler   *labeler.Labeler

	nodeInformer  corev1.NodeInformer
	nplsInformer  informers.NodePoolLabelSetInformer
	workqueue     workqueue.RateLimitingInterface
	clientset     kubernetes.Interface
	nplsClientset npls_clientset.Interface

	logger       log.Logger
	errorHandler emperror.Handler
}

// New gives back an initialized Controller
func New(config Config, k8sConfig *rest.Config, labeler *labeler.Labeler, logger log.Logger, errorHandler emperror.Handler) (*Controller, error) {
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not get k8s clientset")
	}

	nplsClientset, err := npls_clientset.NewForConfig(k8sConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not get k8s npls clientset")
	}

	return &Controller{
		k8sConfig: k8sConfig,
		labeler:   labeler,

		namespace:          config.Namespace,
		nodepoolNameLabels: config.NodepoolNameLabels,

		workqueue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		clientset:     clientset,
		nplsClientset: nplsClientset,

		logger:       logger,
		errorHandler: errorHandler,
	}, nil
}

// Start initializes the informers and starts observing them
func (c *Controller) Start() error {
	stopCh := make(chan struct{})
	defer close(stopCh)

	nodeInformerFactory, nodeInformer := GetNodeInformer(c.clientset, 0, c.workqueue)
	nodeInformerFactory.Start(stopCh)
	c.nodeInformer = nodeInformer

	nplsInformerFactory, nplsInformer := GetNPLSInformer(c.nplsClientset, 0, c.workqueue)
	nplsInformerFactory.Start(stopCh)
	c.nplsInformer = nplsInformer

	err := c.run(10, stopCh)
	if err != nil {
		return errors.WrapIf(err, "could not observe")
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm

	return nil
}

// Run start observing the informers and processing events
func (c *Controller) run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	c.logger.Info("starting NPLS resource controller")

	c.logger.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.nodeInformer.Informer().HasSynced, c.nplsInformer.Informer().HasSynced); !ok {
		return errors.New("failed to wait for caches to sync")
	}

	c.logger.Info("starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	c.logger.Info("workers started")

	<-stopCh
	c.logger.Info("shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var event *Event
		var ok bool
		if event, ok = obj.(*Event); !ok {
			c.workqueue.Forget(obj)
			c.errorHandler.Handle(errors.NewWithDetails("expected string in workqueue", "value", obj))
			return nil
		}

		if err := c.processItem(event); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(event)
			return errors.WrapIfWithDetails(err, "could not sync; requeuing", "key", event.key)
		}

		c.workqueue.Forget(obj)

		return nil
	}(obj)

	if err != nil {
		c.errorHandler.Handle(err)
		return true
	}

	return true
}

func (c *Controller) processItem(event *Event) error {
	var labelsToSet map[string]string
	var err error

	namespace, name, err := cache.SplitMetaNamespaceKey(event.key)
	if err != nil {
		return errors.WrapIfWithDetails(err, "could not split key", "key", event.key)
	}

	if namespace != "" && namespace != c.namespace {
		return nil
	}

	switch event.resourceType {
	case NPLSResourceType:
		var npls *v1alpha1.NodePoolLabelSet
		if event.eventType == AddEvent || event.eventType == UpdateEvent {
			npls, err = c.nplsInformer.Lister().NodePoolLabelSets(namespace).Get(name)
			if err != nil {
				return errors.WrapIfWithDetails(err, "could not get npls from store", "key", event.key)
			}
			labelsToSet = npls.Spec.Labels
		}

		nodes, err := c.getNodesOfANodepool(name)
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not get nodes for a nodepool", "nodepoolName", name)
		}
		for _, node := range nodes {
			err := c.labeler.SyncLabels(&node, labelsToSet)
			if err != nil {
				c.errorHandler.Handle(err)
			}
		}
	case NodeResourceType:
		node, err := c.nodeInformer.Lister().Get(name)
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not get node from store", "node", name)
		}
		npls, err := c.getRelatedNPLSForNode(node)
		if k8serrors.IsNotFound(errors.Cause(err)) {
			err = nil
			npls = nil
		}
		if err != nil {
			return errors.WrapIfWithDetails(err, "could not get related npls for a node", "node", name)
		}
		if npls != nil {
			labelsToSet = npls.Spec.Labels
		}
		err = c.labeler.SyncLabels(node, labelsToSet)
		if err != nil {
			c.errorHandler.Handle(err)
		}
	}
	return nil
}

func (c *Controller) getRelatedNPLSForNode(node *api_v1.Node) (*v1alpha1.NodePoolLabelSet, error) {
	nodepoolName := c.determineNodepoolNameFromNode(node)
	if nodepoolName == "" {
		return nil, nil
	}

	npls, err := c.nplsClientset.LabelsV1alpha1().NodePoolLabelSets(c.namespace).Get(nodepoolName, meta_v1.GetOptions{})
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "could not get npls", "name", nodepoolName)
	}

	return npls, nil
}

func (c *Controller) getNodesOfANodepool(name string) ([]api_v1.Node, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return nil, errors.WrapIf(err, "could not list nodes")
	}

	_nodes := make([]api_v1.Node, 0)
	for _, node := range nodes.Items {
		if c.determineNodepoolNameFromNode(&node) == name {
			_nodes = append(_nodes, node)
		}
	}

	return _nodes, nil
}

func (c *Controller) determineNodepoolNameFromNode(node *api_v1.Node) string {
	labels := node.GetLabels()

	for _, label := range c.nodepoolNameLabels {
		if labels[label] != "" {
			return labels[label]
		}
	}

	return ""
}
