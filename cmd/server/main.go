package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	stdlog "log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/DazWilkin/vultr-exporter/collector"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace string = "vultr"
	subsystem string = "exporter"
	version   string = "v0.0.1"
)
const (
	rootTemplate string = `
{{- define "content" }}
<!DOCTYPE html>
<html lang="en-US">
<head>
<title>Prometheus Exporter for Vultr</title>
<style>
body {
  font-family: Verdana;
}
</style>
</head>
<body>
	<h2>Prometheus Exporter for Vultr</h2>
	<hr/>
	<ul>
	<li><a href="{{ .MetricsPath }}">metrics</a></li>
	<li><a href="/healthz">healthz</a></li>
	</ul>
</body>
</html>
{{- end}}
`
)

var (
	// GitCommit is the git commit value and is expected to be set during build
	GitCommit string
	// GoVersion is the Golang runtime version
	GoVersion = runtime.Version()
	// OSVersion is the OS version (uname --kernel-release) and is expected to be set during build
	OSVersion string
	// StartTime is the start time of the exporter represented as a UNIX epoch
	StartTime = time.Now().Unix()
)
var (
	endpoint    = flag.String("endpoint", "0.0.0.0:8080", "The endpoint of the HTTP server")
	metricsPath = flag.String("path", "/metrics", "The path on which Prometheus metrics will be served")
)
var (
	name string = fmt.Sprintf("%s_%s", namespace, subsystem)
)

type Content struct {
	MetricsPath string
}

func handleHealthz(log logr.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Error(err, "unable to write response")
		}
	}
}
func handleRoot(log logr.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		t := template.Must(template.New("content").Parse(rootTemplate))
		if err := t.ExecuteTemplate(w, "content", Content{MetricsPath: *metricsPath}); err != nil {
			log.Error(err, "unable to execute template")
		}
	}
}
func NewVultrClient(name, key string) *govultr.Client {
	ctx := context.Background()

	config := &oauth2.Config{}
	token := &oauth2.Token{
		AccessToken: key,
	}
	ts := config.TokenSource(ctx, token)
	client := govultr.NewClient(oauth2.NewClient(ctx, ts))

	// Optional changes
	// _ = client.SetBaseURL("https://api.vultr.com")
	client.SetUserAgent(name)
	// client.SetRateLimit(500)
	return client
}
func main() {
	log := stdr.NewWithOptions(stdlog.New(os.Stderr, "", stdlog.LstdFlags), stdr.Options{LogCaller: stdr.All})
	log = log.WithName("main")

	flag.Parse()
	if *endpoint == "" {
		log.Info("Expected flag `--endpoint`")
		os.Exit(1)
	}

	var key string
	if key = os.Getenv("API_KEY"); key == "" {
		log.Info("Expected `API_KEY` in the environment")
		os.Exit(1)
	}

	if GitCommit == "" {
		log.Info("GitCommit value unchanged: expected to be set during build")
	}
	if OSVersion == "" {
		log.Info("OSVersion value unchanged: expected to be set during build")
	}

	// Objects that holds GCP-specific resources (e.g. projects)
	client := NewVultrClient(name, key)

	registry := prometheus.NewRegistry()

	s := collector.System{
		Namespace: namespace,
		Subsystem: subsystem,
		Version:   version,
	}

	b := collector.Build{
		OsVersion: OSVersion,
		GoVersion: GoVersion,
		GitCommit: GitCommit,
		StartTime: StartTime,
	}
	registry.MustRegister(collector.NewExporterCollector(s, b, log))
	registry.MustRegister(collector.NewAccountCollector(s, client, log))
	registry.MustRegister(collector.NewBillingCollector(s, client, log))
	registry.MustRegister(collector.NewBlockStorageCollector(s, client, log))
	registry.MustRegister(collector.NewKubernetesCollector(s, client, log))
	registry.MustRegister(collector.NewLoadBalancerCollector(s, client, log))
	registry.MustRegister(collector.NewReservedIPsCollector(s, client, log))

	mux := http.NewServeMux()
	mux.Handle("/", handleRoot(log))
	mux.Handle("/healthz", handleHealthz(log))
	mux.Handle(*metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Info("Starting",
		"endpoint", *endpoint,
		"metrics", *metricsPath,
	)
	log.Error(http.ListenAndServe(*endpoint, mux), "unable to start server")
}
