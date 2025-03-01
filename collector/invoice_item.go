package collector

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

// InvoiceItemCollector represents a single invoice item type and handles metric aggregation.
// It collects and aggregates metrics for a specific product type, ensuring that multiple
// invoice items of the same type are properly combined before being emitted as Prometheus metrics.
// This prevents duplicate metrics and ensures accurate totals.
type InvoiceItemCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	// Single metric for units that includes unit price as a label
	// This allows for direct calculation of costs using the unit price label
	// and follows Prometheus best practice of having raw values in metrics
	Units *prometheus.Desc

	// Total cost metric
	Total *prometheus.Desc

	// Store aggregated values for the current collection cycle
	// These maps are cleared after metrics are emitted
	currentUnits     map[string]float64 // Maps unit_type to total units
	currentUnitPrice map[string]float64 // Maps unit_type to latest unit price
	currentTotal     float64            // Running total for all items
	description      string             // Description of the item
	product          string             // Product type
}

// NewInvoiceItemCollector creates a new InvoiceItemCollector
// This type does not implement Prometheus's Collector interface
// Because its Collect method additionally takes a *govultr.InvoiceItem
func NewInvoiceItemCollector(s System, client *govultr.Client, log logr.Logger) *InvoiceItemCollector {
	return &InvoiceItemCollector{
		System: s,
		Client: client,
		Log:    log,

		Units: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "units"),
			"Number of units consumed",
			[]string{
				"product",
				"description",
				"unit_type",
				"unit_price", // Added unit price as a label to allow for direct cost calculation
			},
			nil,
		),
		Total: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "cost_usd"),
			"Total cost in USD",
			[]string{
				"product",
				"description",
			},
			nil,
		),

		currentUnits:     make(map[string]float64),
		currentUnitPrice: make(map[string]float64),
	}
}

// Aggregate adds an invoice item's values to the current aggregation
func (c *InvoiceItemCollector) Aggregate(invoiceItem *govultr.InvoiceItem) {
	// Reset aggregated values if this is the first item
	if len(c.currentUnits) == 0 {
		c.currentTotal = 0
	}

	// Store product info
	c.product = invoiceItem.Product
	c.description = invoiceItem.Description

	// Aggregate values
	c.currentUnits[invoiceItem.UnitType] += float64(invoiceItem.Units)
	c.currentUnitPrice[invoiceItem.UnitType] = float64(invoiceItem.UnitPrice) // Use latest price
	c.currentTotal += float64(invoiceItem.Total)
}

// EmitMetrics emits all aggregated metrics
func (c *InvoiceItemCollector) EmitMetrics(ch chan<- prometheus.Metric) {
	// Only emit metrics after aggregating all values
	for unitType, units := range c.currentUnits {
		ch <- prometheus.MustNewConstMetric(
			c.Units,
			prometheus.GaugeValue,
			units,
			[]string{
				c.product,
				c.description,
				unitType,
				fmt.Sprintf("%.6f", c.currentUnitPrice[unitType]), // Format unit price with sufficient precision
			}...,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		c.Total,
		prometheus.GaugeValue,
		c.currentTotal,
		[]string{c.product, c.description}...,
	)

	// Reset the aggregated values
	c.currentUnits = make(map[string]float64)
	c.currentUnitPrice = make(map[string]float64)
	c.currentTotal = 0
	c.product = ""
	c.description = ""
}

// Collect collects metrics
func (c *InvoiceItemCollector) Collect(ch chan<- prometheus.Metric, invoiceItem *govultr.InvoiceItem) {
	c.Aggregate(invoiceItem)
}

// Describe describes metrics
func (c *InvoiceItemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Units
	ch <- c.Total
}

// canonicalize converts a string to lowercase and replaces spaces with underscores
// It is used to convert product names (e.g. "Load Balancer") to a valid label value (e.g. "load_balancer")
func canonicalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}
