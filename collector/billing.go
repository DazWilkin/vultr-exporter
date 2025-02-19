package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

// BillingCollector represents billing related metrics.
// It manages a set of InvoiceItemCollectors, one per product type,
// and ensures that metrics are properly aggregated and deduplicated
// before being emitted to Prometheus.
type BillingCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	// Map of product name to collector to prevent duplicates
	// Each product type (e.g., "Cloud Compute", "Load Balancer") gets its own collector
	// to ensure proper metric aggregation and avoid duplicates
	collectors map[string]*InvoiceItemCollector
}

// NewBillingCollector creates a new BillingCollector
func NewBillingCollector(s System, client *govultr.Client, log logr.Logger) *BillingCollector {
	return &BillingCollector{
		System:     s,
		Client:     client,
		Log:        log,
		collectors: make(map[string]*InvoiceItemCollector),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *BillingCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	invoiceItems, _, err := c.Client.Billing.ListPendingCharges(ctx, &govultr.ListOptions{})
	if err != nil {
		log.Info("Unable to get account details", "error", err)
		return
	}

	// Group invoice items by product to ensure proper aggregation
	itemsByProduct := make(map[string][]*govultr.InvoiceItem)
	for _, item := range invoiceItems {
		itemsByProduct[item.Product] = append(itemsByProduct[item.Product], &item)
	}

	// Clean up collectors for products that no longer exist
	for product := range c.collectors {
		if _, exists := itemsByProduct[product]; !exists {
			delete(c.collectors, product)
		}
	}

	// Process each product's items through its dedicated collector
	for product, items := range itemsByProduct {
		collector, exists := c.collectors[product]
		if !exists {
			collector = NewInvoiceItemCollector(System{
				Namespace: c.System.Namespace,
				Subsystem: fmt.Sprintf("billing_%s", canonicalize(product)),
				Version:   c.System.Version,
			}, c.Client, log)
			c.collectors[product] = collector
		}

		// First aggregate all items for this product to ensure accurate totals
		for _, item := range items {
			collector.Aggregate(item)
		}
		// Then emit the aggregated metrics once per product
		collector.EmitMetrics(ch)
	}
}

// getKeys returns a slice of map keys for logging
func getKeys(m map[string][]*govultr.InvoiceItem) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *BillingCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors {
		collector.Describe(ch)
	}
}
