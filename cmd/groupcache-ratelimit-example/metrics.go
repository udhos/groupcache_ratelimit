package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/udhos/groupcache_exporter"
	"github.com/udhos/groupcache_ratelimit/ratelimit"
)

// metrics is used only to make sure limiter.MetricsExporter conforms with groupcache_exporter.NewExporter.
func metrics(limiter *ratelimit.Limiter) {
	/*
		//
		// expose prometheus metrics
		//
		metricsRoute := "/metrics"
		metricsPort := ":3000"

		log.Printf("starting metrics server at: %s %s", metricsPort, metricsRoute)
	*/

	registry := prometheus.NewRegistry()

	//exporter := modernprogram.New(cache)
	exporter := limiter.MetricsExporter()
	labels := map[string]string{
		//"app": "app1",
	}
	namespace := ""
	collector := groupcache_exporter.NewExporter(namespace, labels, exporter)
	//prometheus.MustRegister(collector)
	registry.MustRegister(collector)

	/*
		go func() {
			http.Handle(metricsRoute, promhttp.Handler())
			log.Fatal(http.ListenAndServe(metricsPort, nil))
		}()
	*/
}
