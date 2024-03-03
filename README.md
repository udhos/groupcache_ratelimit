[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/groupcache_ratelimit/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/groupcache_ratelimit)](https://goreportcard.com/report/github.com/udhos/groupcache_ratelimit)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/groupcache_ratelimit.svg)](https://pkg.go.dev/github.com/udhos/groupcache_ratelimit)

# groupcache_ratelimit

[groupcache_ratelimit](https://github.com/udhos/groupcache_ratelimit) provides rate limiting for distributed applications using [groupcache](https://github.com/modernprogram/groupcache).

# Usage

```go
import (
    "github.com/modernprogram/groupcache/v2"
    "github.com/udhos/groupcache_exporter"
    "github.com/udhos/groupcache_exporter/groupcache/modernprogram"
    "github.com/udhos/groupcache_ratelimit/ratelimit"
    "github.com/udhos/kubegroup/kubegroup"
)

//
// Initialize groupcache as usual.
//

groupcachePort := ":5000"

workspace := groupcache.NewWorkspace()

// Example using kubegroup for kubernetes peers autodiscovery
myURL, errURL = kubegroup.FindMyURL(groupcachePort)

pool := groupcache.NewHTTPPoolOptsWithWorkspace(workspace, myURL, &groupcache.HTTPPoolOptions{})

// Start groupcache peering server
server := &http.Server{Addr: groupcachePort, Handler: pool}
go func() {
    log.Printf("groupcache server: listening on %s", groupcachePort)
    err := server.ListenAndServe()
    log.Printf("groupcache server: exited: %v", err)
}()

// Start kubegroup autodiscovery
optionsKg := kubegroup.Options{
    Pool:           pool,
    GroupCachePort: groupCachePort,
}
group, errGroup := kubegroup.UpdatePeers(optionsKg)
if errGroup != nil {
    log.Fatalf("kubegroup: %v", errGroup)
}

//
// Create the rate limiter: 60 slots at every 30-seconds interval.
//
optionsLim := ratelimit.Options{
    Interval:            30 * time.Second,
    Slots:               60,
    GroupcacheWorkspace: workspace,
}

lim := ratelimit.New(optionsLim)

//
// Optionally expose groupcache metrics for rate limiter in Prometheus format.
//
labels := map[string]string{}
namespace := ""
collector := groupcache_exporter.NewExporter(namespace, labels, lim.MetricsExporter())
prometheus.MustRegister(collector)

//
// Query the rate limiter.
//
accept, errRate := lim.Consume(context.TODO(), "some-key")
if errRate == nil {
    if accept {
        log.Printf("key accepted by rate limiting")
    } else {
        log.Printf("key rejected by rate limiting")
    }
} else {
    log.Printf("rate limiting error: %v", errRate)
}
```

# Istio interceptionMode TPROXY

Istio sidecard interceptionMode with REDIRECT hides real source IP.

In order to receive the real POD source IP to perform rate limiting,
set POD annotation `sidecar.istio.io/interceptionMode` to `TPROXY`.

```yaml
annotations:
  "sidecar.istio.io/interceptionMode": "TPROXY" # REDIRECT or TPROXY
```

Documentation: https://istio.io/latest/docs/reference/config/annotations/#SidecarInterceptionMode

Example helm chart: https://github.com/udhos/kubecache/blob/main/charts/kubecache/values.yaml#L40
