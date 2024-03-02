[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/groupcache_ratelimit/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/groupcache_ratelimit)](https://goreportcard.com/report/github.com/udhos/groupcache_ratelimit)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/groupcache_ratelimit.svg)](https://pkg.go.dev/github.com/udhos/groupcache_ratelimit)

# groupcache_ratelimit

# istio interceptionMode TPROXY

Istio sidecard interceptionMode with REDIRECT hides real source IP.

In order to receive the real POD source IP to perform rate limiting,
set POD annotation `sidecar.istio.io/interceptionMode` to `TPROXY`.

```yaml
annotations:
  "sidecar.istio.io/interceptionMode": "TPROXY" # REDIRECT or TPROXY
```

Documentation: https://istio.io/latest/docs/reference/config/annotations/#SidecarInterceptionMode

Example helm chart: https://github.com/udhos/kubecache/blob/main/charts/kubecache/values.yaml#L40
