package main

import (
	"fmt"
	"k8s.io/client-go/pkg/api/v1"
)

// Endpoint is a summary of kubernetes endpoint
type Endpoint struct {
	Name    string
	Address string
	Port    int32
	RefName string
}

// NewEndpoint allows to create Endpoint
func NewEndpoint(name, address string, port int32, refName string) Endpoint {
	return Endpoint{name, address, port, refName}
}

func generateEntries(endpoint *v1.Endpoints) []Endpoint {
	var (
		eps     []Endpoint
		refName string
	)

	for _, subset := range endpoint.Subsets {
		for _, addr := range subset.Addresses {
			if addr.TargetRef != nil {
				refName = addr.TargetRef.Name
			}
			for _, port := range subset.Ports {
				eps = append(eps, NewEndpoint(endpoint.Name, addr.IP, port.Port, refName))
			}
		}
	}

	return eps
}

func (k2c *kube2consul) updateEndpoints(ep *v1.Endpoints) error {
	endpoints := generateEntries(ep)
	for _, e := range endpoints {
		if err := k2c.registerEndpoint(e); err != nil {
			return fmt.Errorf("Error updating endpoints %v: %v", ep.Name, err)
		}
	}
	if err := k2c.removeDeletedEndpoints(ep.Name, endpoints); err != nil {
		return fmt.Errorf("Error removing possible deleted endpoints: %v", err)
	}
	return nil
}
