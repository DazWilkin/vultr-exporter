package collector

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

// BlockStorageCollector represents Block Storage
type BlockStorageCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger
	Up     *prometheus.Desc
	Block  *prometheus.Desc
}

// NewBlockStorageCollector create a new BlockStorageCollector
func NewBlockStorageCollector(s System, client *govultr.Client, log logr.Logger) *BlockStorageCollector {
	subsystem := "block_storage"
	return &BlockStorageCollector{
		System: s,
		Client: client,
		Log:    log,
		Up: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "up"),
			"Block Storage",
			[]string{
				"label",
				"region",
				"status",
				"block_type",
			},
			nil,
		),
		Block: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "size"),
			"Size of Block Storage",
			[]string{
				"label",
				"region",
				"status",
				"block_type",
			},
			nil,
		),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *BlockStorageCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx := context.Background()

	// Get all block storage volumes across all pages
	var allBlocks []govultr.BlockStorage
	options := &govultr.ListOptions{
		PerPage: 100,
	}

	for {
		blocks, meta, _, err := c.Client.BlockStorage.List(ctx, options)
		if err != nil {
			log.Error(err, "Unable to BlockStorage.List")
			return
		}

		allBlocks = append(allBlocks, blocks...)

		// If we've received all items or there's no next page, break
		if meta != nil && meta.Links != nil && meta.Links.Next == "" {
			break
		}

		// Move to next page
		options.Cursor = meta.Links.Next
	}

	// Enumerate the blocks
	var wg sync.WaitGroup
	for _, block := range allBlocks {
		wg.Add(1)
		go func(block govultr.BlockStorage) {
			defer wg.Done()
			ch <- prometheus.MustNewConstMetric(
				c.Up,
				prometheus.CounterValue,
				func(status string) (result float64) {
					if status == "active" {
						result = 1.0
					}
					return result
				}(block.Status),
				[]string{
					block.Label,
					block.Region,
					block.Status,
					block.BlockType,
				}...,
			)
			ch <- prometheus.MustNewConstMetric(
				c.Block,
				prometheus.GaugeValue,
				float64(block.SizeGB),
				[]string{
					block.Label,
					block.Region,
					block.Status,
					block.BlockType,
				}...,
			)
		}(block)
	}
	wg.Wait()
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *BlockStorageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Up
	ch <- c.Block
}
