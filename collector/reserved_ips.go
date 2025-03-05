package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

// ReservedIPsCollector represents Reserved IPs
type ReservedIPsCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger
	Up     *prometheus.Desc
}

// NewReservedIPsCollector creates a new ResevedIPsCollector
func NewReservedIPsCollector(s System, client *govultr.Client, log logr.Logger) *ReservedIPsCollector {
	subsystem := "reserved_ips"
	return &ReservedIPsCollector{
		System: s,
		Client: client,
		Log:    log,
		Up: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "up"),
			"Reserved IPs",
			[]string{
				"region",
				"type",
				"subnet_size",
				"label",
			},
			nil,
		),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *ReservedIPsCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx := context.Background()

	// Get all reserved IPs across all pages
	var allIPs []govultr.ReservedIP
	options := &govultr.ListOptions{
		PerPage: 100,
	}

	for {
		ips, meta, _, err := c.Client.ReservedIP.List(ctx, options)
		if err != nil {
			log.Error(err, "Unable to List")
			return
		}

		allIPs = append(allIPs, ips...)

		// If we've received all items or there's no next page, break
		if meta != nil && meta.Links != nil && meta.Links.Next == "" {
			break
		}

		// Move to next page
		options.Cursor = meta.Links.Next
	}

	// Enumerate the IPs
	var wg sync.WaitGroup
	for _, ip := range allIPs {
		wg.Add(1)
		go func(ip govultr.ReservedIP) {
			defer wg.Done()

			ch <- prometheus.MustNewConstMetric(
				c.Up,
				prometheus.CounterValue,
				1.0,
				[]string{
					ip.Region,
					ip.IPType,
					fmt.Sprintf("%d", (ip.SubnetSize)),
					ip.Label,
				}...,
			)
		}(ip)
	}
	wg.Wait()
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *ReservedIPsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Up
}
