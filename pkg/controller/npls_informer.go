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
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	clientset "github.com/banzaicloud/nodepool-labels-operator/pkg/client/clientset/versioned"
	informers "github.com/banzaicloud/nodepool-labels-operator/pkg/client/informers/externalversions"
	v1alpha "github.com/banzaicloud/nodepool-labels-operator/pkg/client/informers/externalversions/nodepoollabelset/v1alpha1"
)

const (
	NPLSResourceType = "npls"
)

// GetNPLSInformer creates and gives back a shared NPLS informer and its factory
func GetNPLSInformer(clientset clientset.Interface, resync time.Duration, queue workqueue.RateLimitingInterface) (informers.SharedInformerFactory, v1alpha.NodePoolLabelSetInformer) {
	factory := informers.NewSharedInformerFactory(clientset, resync)
	informer := factory.Labels().V1alpha1().NodePoolLabelSets()

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(old)
			if err == nil {
				queue.Add(NewEvent(NPLSResourceType, UpdateEvent, key))
			}
		},
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(NewEvent(NPLSResourceType, AddEvent, key))
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(NewEvent(NPLSResourceType, DeleteEvent, key))
			}
		},
	})

	return factory, informer
}
