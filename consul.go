package main

import (
	"fmt"
	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"
)

func newConsulClient(consulAPI, consulToken string) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	config.Address = consulAPI
	config.Token = consulToken

	consulClient, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	glog.Infof("Testing communication with consul server")
	_, err = consulClient.Status().Leader()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with consul server: %v", err)
	}
	glog.Infof("Communication with consul server successful")

	return consulClient, nil
}

func (k2c *kube2consul) registerEndpoint(e Endpoint) {
	if e.RefName == "" {
		return
	}

	service := &consulapi.AgentService{
		Service: e.Name,
		Port:    int(e.Port),
		Tags:    []string{consulTag},
	}

	reg := &consulapi.CatalogRegistration{
		Node:    e.RefName,
		Address: e.Address,
		Service: service,
	}

	_, err := k2c.consulCatalog.Register(reg, nil)
	if err != nil {
		glog.Errorf("Error registrating service %v (%v, %v): %v", e.Name, e.RefName, e.Address, err)
	} else {
		glog.V(1).Infof("Update service %v (%v, %v)", e.Name, e.RefName, e.Address)
	}
}

func endpointExists(refName, address string, port int, endpoints []Endpoint) bool {
	for _, e := range endpoints {
		if e.RefName == refName && e.Address == address && int(e.Port) == port {
			return true
		}
	}
	return false
}

func (k2c *kube2consul) removeDeletedEndpoints(serviceName string, endpoints []Endpoint) {
	updatedNodes := make(map[string]struct{})
	services, _, err := k2c.consulCatalog.Service(serviceName, consulTag, nil)
	if err != nil {
		glog.Errorf("[Consul] Failed to get services: %v", err)
		return
	}

	for _, service := range services {
		if !endpointExists(service.Node, service.Address, service.ServicePort, endpoints) {
			dereg := &consulapi.CatalogDeregistration{
				Node:      service.Node,
				Address:   service.Address,
				ServiceID: service.ServiceID,
			}
			_, err := k2c.consulCatalog.Deregister(dereg, nil)
			if err != nil {
				glog.Errorf("Error deregistrating service {node: %s, service: %s, address: %s}: %v", service.Node, service.ServiceName, service.Address, err)
			} else {
				glog.Infof("Deregister service {node: %s, service: %s, address: %s}", service.Node, service.ServiceName, service.Address)
				updatedNodes[service.Node] = struct{}{}
			}
		}
	}

	// Remove all empty nodes
	for nodeName := range updatedNodes {
		if node, _, err := k2c.consulCatalog.Node(nodeName, nil); err != nil {
			glog.Errorf("Cannot get node %s: %v", nodeName, err)
		} else if node != nil && len(node.Services) == 0 {
			dereg := &consulapi.CatalogDeregistration{
				Node: nodeName,
			}
			_, err = k2c.consulCatalog.Deregister(dereg, nil)
			if err != nil {
				glog.Errorf("Error deregistrating node %s: %v", nodeName, err)
			} else {
				glog.Infof("Deregister empty node %s", nodeName)
			}
		}
	}
}
