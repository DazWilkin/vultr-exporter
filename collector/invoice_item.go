package collector

import (
	"strings"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

// InvoiceItemCollector represents a single invoice item
type InvoiceItemCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	Units     *prometheus.Desc
	UnitPrice *prometheus.Desc
	Totals    *prometheus.Desc
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
	}
}

// Collect collects metrics
func (c *InvoiceItemCollector) Collect(ch chan<- prometheus.Metric, invoiceItem *govultr.InvoiceItem) {
	ch <- prometheus.MustNewConstMetric(
		c.Units,
		prometheus.GaugeValue,
		float64(invoiceItem.Units),
		[]string{
			invoiceItem.UnitType,
		}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.UnitPrice,
		prometheus.GaugeValue,
		float64(invoiceItem.UnitPrice),
		[]string{
			invoiceItem.UnitType,
		}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.Totals,
		prometheus.GaugeValue,
		float64(invoiceItem.Total),
		[]string{}...,
	)

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
