package main

import (
	"time"

	"github.com/golang/glog"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	kcache "k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kframework "k8s.io/kubernetes/pkg/controller/framework"
	kselector "k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/util/wait"
)

// Returns a cache.ListWatch that gets all changes to endpoints.
func createEndpointsLW(kubeClient *kclient.Client) *kcache.ListWatch {
	return kcache.NewListWatchFromClient(kubeClient, "endpoints", kapi.NamespaceAll, kselector.Everything())
}

func newKubeClient(kubeAPI string) (*kclient.Client, error) {
	var (
		config *restclient.Config
	)

	config = &restclient.Config{
		Host:          kubeAPI,
		ContentConfig: restclient.ContentConfig{GroupVersion: &unversioned.GroupVersion{Version: "v1"}},
	}

	glog.Infof("Using %s for kubernetes master", config.Host)
	glog.Infof("Using kubernetes API %v", config.GroupVersion)
	return kclient.New(config)
}

func (k2c *kube2consul) handleEndpointUpdate(obj interface{}) {
	if e, ok := obj.(*kapi.Endpoints); ok {
		k2c.updateEndpoints(e)
	}
}

func (k2c *kube2consul) watchEndpoints(kubeClient *kclient.Client) kcache.Store {
	eStore, eController := kframework.NewInformer(
		createEndpointsLW(kubeClient),
		&kapi.Endpoints{},
		time.Duration(opts.resyncPeriod)*time.Second,
		kframework.ResourceEventHandlerFuncs{
			AddFunc: func(newObj interface{}) {
				go k2c.handleEndpointUpdate(newObj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				go k2c.handleEndpointUpdate(newObj)
			},
		},
	)

	go eController.Run(wait.NeverStop)
	return eStore
}
