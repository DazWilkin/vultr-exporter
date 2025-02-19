package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

// BillingCollector represents billing related metrics
type BillingCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	InvoiceItemCollectors []*InvoiceItemCollector
}

// NewBillingCollector creates a new BillingCollector
func NewBillingCollector(s System, client *govultr.Client, log logr.Logger) *BillingCollector {
	// subsystem := "billing"
	return &BillingCollector{
		System: s,
		Client: client,
		Log:    log,

		// Unable to determine size of PendingCharges until we've fetched the data
		InvoiceItemCollectors: []*InvoiceItemCollector{},
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *BillingCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	invoiceItems, _, err := c.Client.Billing.ListPendingCharges(ctx, &govultr.ListOptions{})
	if err != nil {
		log.Info("Unable to get account details")
		return
	}

	log.Info("Response",
		"billing", invoiceItems,
	)

	// Now that we have the data, we can create the metrics and collect them
	// These results must replace any prior PendingCharges
	c.InvoiceItemCollectors = make([]*InvoiceItemCollector, len(invoiceItems))
	for i, invoiceItem := range invoiceItems {
		// Create
		// Must be added to this Collector's InvoiceItemCollectors slice
		c.InvoiceItemCollectors[i] = NewInvoiceItemCollector(System{
			Namespace: c.System.Namespace,
			Subsystem: fmt.Sprintf("%s_%s", c.System.Subsystem, canonicalize(invoiceItem.Product)),
			Version:   c.System.Version,
		}, c.Client, log)

		// Collect
		c.InvoiceItemCollectors[i].Collect(ch, &invoiceItem)
	}
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *BillingCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, invoiceItem := range c.InvoiceItemCollectors {
		invoiceItem.Describe(ch)
	}
}
