package main

import (
	"flag"
	"fmt"
	"github.com/lukasbonny/vodafone-station-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"net/http"
	"os"
	"reflect"
)

const version = "0.0.1"

var (
	showVersion             = flag.Bool("version", false, "Print version and exit")
	showMetrics             = flag.Bool("show-metrics", false, "Show available metrics and exit")
	listenAddress           = flag.String("web.listen-address", "[::]:9420", "Address to listen on")
	metricsPath             = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
	logLevel                = flag.String("log.level", "info", "Logging level")
	vodafoneStationUrl      = flag.String("vodafone.station-url", "http://192.168.0.1", "Vodafone station URL. For bridge mode this is 192.168.100.1 (note: Configure a route if using bridge mode)")
	vodafoneStationPassword = flag.String("vodafone.station-password", "How is the default password calculated? mhmm", "Password for logging into the Vodafone station")
)

func main() {
	flag.Parse()
	err := log.Base().SetLevel(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(2)
	}

	if *showMetrics {
		describeMetrics()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Println("vodafone-station-exporter")
		fmt.Printf("Version: %s\n", version)
		fmt.Println("Author: @fluepke")
		fmt.Println("Prometheus Exporter for the Vodafone Station (CGA4233DE)")
		os.Exit(0)
	}

	startServer()
}

func describeMetrics() {
	fmt.Println("Exported metrics")
	c := &collector.Collector{}
	ch := make(chan *prometheus.Desc)
	go func() {
		defer close(ch)
		c.Describe(ch)
	}()
	for desc := range ch {
		if desc == nil {
			continue
		}
		describeMetric(desc)
	}
}

func describeMetric(desc *prometheus.Desc) {
	fqName := reflect.ValueOf(desc).Elem().FieldByName("fqName").String()
	help := reflect.ValueOf(desc).Elem().FieldByName("help").String()
	labels := reflect.ValueOf(desc).Elem().FieldByName("variableLabels")
	fmt.Println("  * `" + fqName + "`: " + help)
	if labels.Len() == 0 {
		return
	}
	fmt.Print("    - Labels: ")
	first := true
	for i := 0; i < labels.Len(); i++ {
		if !first {
			fmt.Print(", ")
		}
		first = false
		fmt.Print("`" + labels.Index(i).String() + "`")
	}
	fmt.Println("")
}

func startServer() {
	log.Infof("Starting vodafone-station-exporter (version %s)", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>vodafone-station-exporter (Version ` + version + `)</title></head>
            <body>
            <h1>vodafone-station-exporter</h1>
            <a href="/metrics">metrics</a>
            </body>
            </html>`))
	})
	http.HandleFunc(*metricsPath, handleMetricsRequest)

	log.Infof("Listening on %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func handleMetricsRequest(w http.ResponseWriter, request *http.Request) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(&collector.Collector{
		Station: collector.NewVodafoneStation(*vodafoneStationUrl, *vodafoneStationPassword),
	})
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog:      log.NewErrorLogger(),
		ErrorHandling: promhttp.ContinueOnError,
	}).ServeHTTP(w, request)
}
