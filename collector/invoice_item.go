package collector

import (
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

	Units     *prometheus.Desc
	UnitPrice *prometheus.Desc
	Totals    *prometheus.Desc

	// Store aggregated values for the current collection cycle
	// These maps are cleared after metrics are emitted
	currentUnits     map[string]float64 // Maps unit_type to total units
	currentUnitPrice map[string]float64 // Maps unit_type to latest unit price
	currentTotal     float64            // Running total for all items
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
			"Units",
			[]string{
				"unit_type",
			},
			nil,
		),
		UnitPrice: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "unit_price"),
			"Unit Price",
			[]string{
				"unit_type",
			},
			nil,
		),
		Totals: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "total"),
			"Total",
			[]string{},
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
			[]string{unitType}...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.UnitPrice,
			prometheus.GaugeValue,
			c.currentUnitPrice[unitType],
			[]string{unitType}...,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		c.Totals,
		prometheus.GaugeValue,
		c.currentTotal,
		[]string{}...,
	)

	// Reset the aggregated values
	c.currentUnits = make(map[string]float64)
	c.currentUnitPrice = make(map[string]float64)
	c.currentTotal = 0
}

// Collect collects metrics
func (c *InvoiceItemCollector) Collect(ch chan<- prometheus.Metric, invoiceItem *govultr.InvoiceItem) {
	c.Aggregate(invoiceItem)
}

// Describe describes metrics
func (c *InvoiceItemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Units
	ch <- c.UnitPrice
	ch <- c.Totals
}

// canonicalize converts a string to lowercase and replaces spaces with underscores
// It is used to convert product names (e.g. "Load Balancer") to a valid label value (e.g. "load_balancer")
func canonicalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}
