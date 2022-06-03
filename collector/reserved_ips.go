package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v2"
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
	options := &govultr.ListOptions{}
	ips, meta, err := c.Client.ReservedIP.List(ctx, options)
	if err != nil {
		log.Info("Unable to List")
		return
	}

	log.Info("Response",
		"meta", meta,
	)

	// Enumerate the IPs
	var wg sync.WaitGroup
	for _, ip := range ips {
		wg.Add(1)
		go func(ip govultr.ReservedIP) {
			defer wg.Done()
			log.Info("Details",
				"ReservedIP", ip,
			)

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
