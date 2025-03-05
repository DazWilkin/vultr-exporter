package collector

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

// LoadBalancerCollector represents Load Balancers
type LoadBalancerCollector struct {
	System    System
	Client    *govultr.Client
	Log       logr.Logger
	Up        *prometheus.Desc
	Instances *prometheus.Desc
}

// NewLoadBalancerCollector creates a new LoadBalancerCollector
func NewLoadBalancerCollector(s System, client *govultr.Client, log logr.Logger) *LoadBalancerCollector {
	subsystem := "load_balancer"
	return &LoadBalancerCollector{
		System: s,
		Client: client,
		Log:    log,
		Up: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "up"),
			"Load balancer",
			[]string{
				"label",
				"region",
				"status",
			},
			nil,
		),
		Instances: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "instances"),
			"Number of Load balancer instances",
			[]string{
				"label",
				"region",
				"status",
			},
			nil,
		),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *LoadBalancerCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx := context.Background()

	// Get all load balancers across all pages
	var allLoadBalancers []govultr.LoadBalancer
	options := &govultr.ListOptions{
		PerPage: 100,
	}

	for {
		loadbalancers, meta, _, err := c.Client.LoadBalancer.List(ctx, options)
		if err != nil {
			log.Error(err, "Unable to list LoadBalancers")
			return
		}

		allLoadBalancers = append(allLoadBalancers, loadbalancers...)

		// If we've received all items or there's no next page, break
		if meta != nil && meta.Links != nil && meta.Links.Next == "" {
			break
		}

		// Move to next page
		options.Cursor = meta.Links.Next
	}

	// Enumerate all of the loadbalancers
	var wg sync.WaitGroup
	for _, loadbalancer := range allLoadBalancers {
		wg.Add(1)
		go func(lb govultr.LoadBalancer) {
			defer wg.Done()

			ch <- prometheus.MustNewConstMetric(
				c.Up,
				prometheus.CounterValue,
				func(status string) (result float64) {
					if status == "active" {
						result = 1.0
					}
					return result
				}(lb.Status),
				[]string{
					lb.Label,
					lb.Region,
					lb.Status,
				}...,
			)
			ch <- prometheus.MustNewConstMetric(
				c.Instances,
				prometheus.GaugeValue,
				float64(len(lb.Instances)),
				[]string{
					lb.Label,
					lb.Region,
					lb.Status,
				}...,
			)
		}(loadbalancer)
	}
	wg.Wait()
}

// Describe implements Prometheus' Collector interface and is used to Describe metrics
func (c *LoadBalancerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Up
	ch <- c.Instances
}
