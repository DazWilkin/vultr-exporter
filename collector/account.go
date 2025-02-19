package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

var (
	_ prometheus.Collector = (*AccountCollector)(nil)
)

// AccountCollector represents Account
type AccountCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	// BandwidthCollector is a nested collector
	// It is used to collect bandwidth metrics
	// These comprise repeated metrics representing different periods of time
	BandwidthCollector BandwidthCollector

	Balance        *prometheus.Desc
	PendingCharges *prometheus.Desc
}

// NewAccountCollector create a new AccountCollector
func NewAccountCollector(s System, client *govultr.Client, log logr.Logger) *AccountCollector {
	subsystem := "account"

	// BandwidthCollector needs a uniquely named subsystem to differentiate its metrics
	bandwidth := NewBandwidthCollector(System{
		Namespace: s.Namespace,
		Subsystem: fmt.Sprintf("%s_%s", subsystem, "bandwidth"),
		Version:   s.Version,
	}, client, log)

	return &AccountCollector{
		System: s,
		Client: client,
		Log:    log,

		BandwidthCollector: bandwidth,

		Balance: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "balance"),
			"Account Balance",
			[]string{
				"name",
				"email",
			},
			nil,
		),
		PendingCharges: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "pending_charges"),
			"Pending Charges",
			[]string{
				"name",
				"email",
			},
			nil,
		),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *AccountCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx := context.Background()

	var wg sync.WaitGroup

	// Get Account details
	wg.Add(1)
	go func() {
		defer wg.Done()

		account, _, err := c.Client.Account.Get(ctx)
		if err != nil {
			log.Info("Unable to get account details")
			return
		}

		log.Info("Response",
			"account", account,
		)

		ch <- prometheus.MustNewConstMetric(
			c.Balance,
			prometheus.GaugeValue,
			float64(account.Balance),
			[]string{
				account.Name,
				account.Email,
			}...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.PendingCharges,
			prometheus.GaugeValue,
			float64(account.PendingCharges),
			[]string{
				account.Name,
				account.Email,
			}...,
		)
	}()

	// Get Account Bandwidth details
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Collect Bandwidth metrics
		c.BandwidthCollector.Collect(ch)
	}()

	wg.Wait()
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *AccountCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Balance
	ch <- c.PendingCharges

	// Describe Bandwidth metrics
	c.BandwidthCollector.Describe(ch)
}
