package collector

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v2"
)

// BillingCollector represents billing related metrics
type BillingCollector struct {
	System         System
	Client         *govultr.Client
	Log            logr.Logger
	PendingCharges *prometheus.Desc
	Balance        *prometheus.Desc
}

// NewBillingCollector creates a new BillingCollector
func NewBillingCollector(s System, client *govultr.Client, log logr.Logger) *BillingCollector {
	subsystem := "billing"
	return &BillingCollector{
		System: s,
		Client: client,
		Log:    log,
		PendingCharges: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "pending_charges"),
			"Pending Charges",
			nil,
			nil,
		),
		Balance: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "balance"),
			"Account Balance",
			nil,
			nil,
		),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *BillingCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx := context.Background()
	account, err := c.Client.Account.Get(ctx)
	if err != nil {
		log.Info("Unable to get account details")
		return
	}

	log.Info("Response",
		"account", account,
	)

	ch <- prometheus.MustNewConstMetric(
		c.PendingCharges,
		prometheus.CounterValue,
		float64(account.PendingCharges),
	)
	ch <- prometheus.MustNewConstMetric(
		c.Balance,
		prometheus.CounterValue,
		float64(account.Balance),
	)
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *BillingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.PendingCharges
	ch <- c.Balance
}
