package collector

import (
	"fmt"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-logr/stdr"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/vultr/govultr/v3"
)

// Copied from govultr/v3/govultr_test.go
// Creates a mock server and client
// The mux can be used to define the server's responses (see `TestBillingCollector`)
var (
	mux *http.ServeMux
	// ctx    = context.TODO()
	client *govultr.Client
	server *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client = govultr.NewClient(nil)
	thisURL, _ := url.Parse(server.URL)
	client.BaseURL = thisURL
}

func teardown() {
	server.Close()
}

const (
	// Namespace and Subsystem form part (!) of the prefix of metric names
	// Changing these will change the metrics names in the Prometheus example output (below)
	tNamespace string = "test"
	tSubsystem string = "billing"
	tVersion   string = "0.0.1"
)

var (
	// product ("Load Balancer") is canonicalized to "load_balancer"
	// and used by the Collector to form metric names
	// Channging the product value below will change the Prometheus example output (below)
	responsePendingCharges string = `{
		"pending_charges": [
			{
				"description": "Load Balancer (my-loadbalancer)",
				"start_date": "2020-10-10T01:56:20+00:00",
				"end_date": "2020-10-10T01:56:20+00:00",
				"units": 720,
				"unit_type": "hours",
				"unit_price": 0.0149,
				"total": 10,
				"product": "Load Balancer"
			}
		]
	}`
	// The Prometheus output below is generated from the responsePendingCharges above
	// The values are hard-coded and will need to be updated if:
	// Either the Collector is changed (e.g. labels added|removed)
	// Or the responsePendingCharges changes
	prometheusPendingCharges string = `
	# HELP test_billing_load_balancer_total Total
	# TYPE test_billing_load_balancer_total gauge
	test_billing_load_balancer_total 10
	# HELP test_billing_load_balancer_unit_price Unit Price
	# TYPE test_billing_load_balancer_unit_price gauge
	test_billing_load_balancer_unit_price{unit_type="hours"} 0.01489999983459711
	# HELP test_billing_load_balancer_units Units
	# TYPE test_billing_load_balancer_units gauge
	test_billing_load_balancer_units{unit_type="hours"} 720
	`
)

func TestBillingCollector(t *testing.T) {
	setup()
	defer teardown()

	// Overrides the response that the client receives when it calls Billing.ListPendingCharges
	mux.HandleFunc("/v2/billing/pending-charges", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, responsePendingCharges)
	})

	log := stdr.NewWithOptions(stdlog.New(os.Stderr, "", stdlog.LstdFlags), stdr.Options{LogCaller: stdr.All})
	log = log.WithName("test")

	s := System{
		Namespace: tNamespace,
		Subsystem: tSubsystem,
		Version:   tVersion,
	}

	// Client is defined in the global namespace using the govultr test client
	collector := NewBillingCollector(s, client, log)

	// Effectively got=collector and want=prometheusPendingCharges
	if err := testutil.CollectAndCompare(
		collector,
		strings.NewReader(prometheusPendingCharges),
	); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}
