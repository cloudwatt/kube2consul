package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/coreos/pkg/flagutil"
	"github.com/golang/glog"

	consulapi "github.com/hashicorp/consul/api"

	"k8s.io/client-go/pkg/api/v1"
	kcache "k8s.io/client-go/tools/cache"

	health "github.com/docker/go-healthcheck"
)

var (
	opts               cliOpts
	kube2consulVersion string
	lock               *consulapi.Lock
	lockCh             <-chan struct{}
)

const (
	consulTag = "kube2consul"
)

type kube2consul struct {
	consulCatalog  *consulapi.Catalog
	endpointsStore kcache.Store
}

type cliOpts struct {
	kubeAPI      string
	consulAPI    string
	consulToken  string
	resyncPeriod int
	version      bool
	kubeConfig   string
	lock         bool
	lockKey      string
	noHealth     bool
}

func init() {
	flag.BoolVar(&opts.version, "version", false, "Prints kube2consul version")
	flag.IntVar(&opts.resyncPeriod, "resync-period", 30, "Resynchronization period in second")
	flag.StringVar(&opts.kubeAPI, "kubernetes-api", "", "Overrides apiserver address when used in cluster")
	flag.StringVar(&opts.consulAPI, "consul-api", "127.0.0.1:8500", "Consul API URL")
	flag.StringVar(&opts.consulToken, "consul-token", "", "Consul API token")
	flag.StringVar(&opts.kubeConfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	flag.BoolVar(&opts.lock, "lock", false, "Acquires a lock with consul to ensure that only one instance of kube2consul is running")
	flag.StringVar(&opts.lockKey, "lock-key", "locks/kube2consul/.lock", "Key used for locking")
	flag.BoolVar(&opts.noHealth, "no-health", false, "Disable endpoint /health on port 8080")
}

func inSlice(value string, slice []string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

func (k2c *kube2consul) RemoveDNSGarbage() {
	epSet := make(map[string]struct{})

	for _, obj := range k2c.endpointsStore.List() {
		if ep, ok := obj.(*v1.Endpoints); ok {
			epSet[ep.Name] = struct{}{}
		}
	}

	services, _, err := k2c.consulCatalog.Services(nil)
	if err != nil {
		glog.Errorf("Cannot remove DNS garbage: %v", err)
		return
	}

	for name, tags := range services {
		if !inSlice(consulTag, tags) {
			continue
		}

		if _, ok := epSet[name]; !ok {
			k2c.removeDeletedEndpoints(name, []Endpoint{})
		}
	}
}

func consulCheck() error {
	_, err := newConsulClient(opts.consulAPI, opts.consulToken)
	if err != nil {
		return err
	}

	return nil
}
func kubernetesCheck() error {
	_, err := newKubeClient(opts.kubeAPI, opts.kubeConfig)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	// parse flags
	flag.Parse()
	flagutil.SetFlagsFromEnv(flag.CommandLine, "K2C")

	if opts.version {
		fmt.Println(kube2consulVersion)
		os.Exit(0)
	}

	// create consul client
	consulClient, err := newConsulClient(opts.consulAPI, opts.consulToken)
	if err != nil {
		glog.Fatalf("Failed to create a consul client: %v", err)
	}

	// create kubernetes client
	kubeClient, err := newKubeClient(opts.kubeAPI, opts.kubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create a kubernetes client: %v", err)
	}

	if !opts.noHealth {
		health.RegisterPeriodicThresholdFunc("consul", time.Second*5, 3, consulCheck)
		health.RegisterPeriodicThresholdFunc("kubernetes", time.Second*5, 3, kubernetesCheck)
		go func() {
			// create http server to expose health status
			r := mux.NewRouter()
			r.HandleFunc("/health", health.StatusHandler)
			srv := &http.Server{
				Handler:     r,
				Addr:        "0.0.0.0:8080",
				ReadTimeout: 15 * time.Second,
			}
			glog.Fatal(srv.ListenAndServe())
		}()
	}

	if opts.lock {
		glog.Info("Attempting to acquire lock")
		lock, err = consulClient.LockKey(opts.lockKey)
		if err != nil {
			glog.Fatalf("Lock setup failed :%v", err)
		}
		stopCh := make(chan struct{})
		lockCh, err = lock.Lock(stopCh)
		if err != nil {
			glog.Fatalf("Failed acquiring lock: %v", err)
		}
		glog.Info("Lock acquired")
	}

	k2c := kube2consul{
		consulCatalog: consulClient.Catalog(),
	}

	k2c.endpointsStore = k2c.watchEndpoints(kubeClient)

	// Handle SIGINT and SIGTERM.
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-time.NewTicker(time.Duration(opts.resyncPeriod) * time.Second).C:
			k2c.RemoveDNSGarbage()
		case <-lockCh:
			glog.Fatalf("Lost lock, Exting")
		case sig := <-sigs:
			glog.Infof("Recieved signal: %v", sig)
			if opts.lock {
				// Release the lock before termination
				glog.Infof("Attempting to release lock")
				if err := lock.Unlock(); err != nil {
					glog.Errorf("Lock release failed : %s", err)
					os.Exit(1)
				}
				glog.Infof("Lock released")
				// Cleanup the lock if no longer in use
				glog.Infof("Cleaning lock entry")
				if err := lock.Destroy(); err != nil {
					if err != consulapi.ErrLockInUse {
						glog.Errorf("Lock cleanup failed: %s", err)
						os.Exit(1)
					} else {
						glog.Infof("Cleanup aborted, lock in use")
					}
				} else {
					glog.Infof("Cleanup succeeded")
				}
				glog.Infof("Exiting")
			}
			os.Exit(0)
		}
	}
}
