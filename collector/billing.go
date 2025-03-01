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
// It manages a set of InvoiceItemCollectors, one per product type and description,
// and ensures that metrics are properly aggregated and deduplicated
// before being emitted to Prometheus.
type BillingCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	// Map of product+description to collector to prevent duplicates
	// Each unique product instance (e.g., different Load Balancers) gets its own collector
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

// getCollectorKey generates a unique key for a collector based on product and description
func getCollectorKey(product, description string) string {
	return fmt.Sprintf("%s::%s", product, description)
}

// getAllPendingCharges retrieves all pending charges across all pages
func (c *BillingCollector) getAllPendingCharges(ctx context.Context) ([]govultr.InvoiceItem, error) {
	var allItems []govultr.InvoiceItem
	options := &govultr.ListOptions{
		PerPage: 500,
	}

	for {
		items, _, err := c.Client.Billing.ListPendingCharges(ctx, options)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		// If we got less items than requested, we've reached the end
		if len(items) < options.PerPage {
			break
		}

		// Move to next page
		options.Cursor = fmt.Sprintf("%d", len(allItems))
	}

	return allItems, nil
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *BillingCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Increased timeout for pagination
	defer cancel()

	// Get all pending charges across all pages
	invoiceItems, err := c.getAllPendingCharges(ctx)
	if err != nil {
		c.Log.Error(err, "Unable to get account details")
		return
	}

	// Group invoice items by product+description to ensure proper aggregation
	itemsByKey := make(map[string][]*govultr.InvoiceItem)
	for i := range invoiceItems {
		item := &invoiceItems[i] // Get pointer to item in slice to avoid copying
		key := getCollectorKey(item.Product, item.Description)
		itemsByKey[key] = append(itemsByKey[key], item)
	}

	// Clean up collectors for products that no longer exist
	for key := range c.collectors {
		if _, exists := itemsByKey[key]; !exists {
			delete(c.collectors, key)
		}
	}

	// Process each product's items through its dedicated collector
	for key, items := range itemsByKey {
		collector, exists := c.collectors[key]
		if !exists {
			collector = NewInvoiceItemCollector(System{
				Namespace: c.System.Namespace,
				Subsystem: "billing",
				Version:   c.System.Version,
			}, c.Client, c.Log)
			c.collectors[key] = collector
		}

		// First aggregate all items for this product to ensure accurate totals
		for _, item := range items {
			collector.Aggregate(item)
		}
		// Then emit the aggregated metrics once per product
		collector.EmitMetrics(ch)
	}
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *BillingCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors {
		collector.Describe(ch)
	}
}
