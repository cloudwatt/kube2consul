package main

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func newKubeClient(apiserver string, kubeconfig string) (kubeClient kubernetes.Interface, err error) {
	if kubeconfig == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		// Allow overriding of apiserver if using inClusterConfig
		// (necessary if kube-proxy isn't properly set up).
		if apiserver != "" {
			config.Host = apiserver
		}
		tokenPresent := false
		if len(config.BearerToken) > 0 {
			tokenPresent = true
		}
		glog.Infof("service account token present: %v", tokenPresent)
		glog.Infof("service host: %s", config.Host)
		if kubeClient, err = kubernetes.NewForConfig(config); err != nil {
			return nil, err
		}
	} else {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		// if you want to change the loading rules (which files in which order), you can do so here
		loadingRules.ExplicitPath = kubeconfig
		configOverrides := &clientcmd.ConfigOverrides{}
		// if you want to change override values or bind them to flags, there are methods to help you
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := kubeConfig.ClientConfig()
		//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		//config, err := clientcmd.DefaultClientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
		kubeClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	glog.Infof("Testing communication with k8s apiserver")
	_, err = kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with k8s apiserver: %v", err)
	}
	glog.Infof("Communication with k8s apiserver successful")

	return kubeClient, nil
}

// Returns a cache.ListWatch that gets all changes to endpoints.
func createEndpointsListWatcher(kubeClient kubernetes.Interface) *kcache.ListWatch {
	client := kubeClient.CoreV1().RESTClient()
	return kcache.NewListWatchFromClient(client, "endpoints", kapi.NamespaceAll, nil)
}

func (k2c *kube2consul) handleEndpointUpdate(obj interface{}) {
	if e, ok := obj.(*v1.Endpoints); ok {
		k2c.updateEndpoints(e)
	}
}

func (k2c *kube2consul) watchEndpoints(kubeClient kubernetes.Interface) kcache.Store {
	eStore, eController := kcache.NewInformer(
		createEndpointsListWatcher(kubeClient),
		&v1.Endpoints{},
		time.Duration(opts.resyncPeriod)*time.Second,
		kcache.ResourceEventHandlerFuncs{
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
