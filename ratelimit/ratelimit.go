// Package ratelimit implements rate limiting with groupcache.
package ratelimit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/modernprogram/groupcache/v2"
	"github.com/udhos/groupcache_exporter/groupcache/modernprogram"
)

// Options define parameters for rate limiting.
type Options struct {
	// Interval is the measurement interval.
	// For example, if Interval is 10s, and Slots is 20,
	// the rate limiter will accept 20 requests at every
	// 10 seconds.
	Interval time.Duration

	// Slots is the amount of requests the rate limiter
	// accepts per Interval.
	// For example, if Interval is 10s, and Slots is 20,
	// the rate limiter will accept 20 requests at every
	// 10 seconds.
	Slots int

	// GroupcacheWorkspace is required groupcache workspace.
	GroupcacheWorkspace *groupcache.Workspace

	// GroupcacheName gives a unique cache name. If unspecified, defaults to rate-limit.
	GroupcacheName string

	// GroupcacheSizeBytes limits the cache size. If unspecified, defaults to 10MB.
	GroupcacheSizeBytes int64
}

// Limiter implements rate limiting.
type Limiter struct {
	options Options
	group   *groupcache.Group
	expire  map[string]time.Time
}

// DefaultGroupcacheSizeBytes is default for unspecified Options GroupcacheSizeBytes.
const DefaultGroupcacheSizeBytes = 10_000_000

// New creates a rate limiter.
func New(options Options) *Limiter {
	if options.Interval < 1 {
		panic("interval must be greater than zero")
	}
	if options.Slots < 1 {
		panic("slots must be greater than zero")
	}
	if options.GroupcacheWorkspace == nil {
		panic("groupcache workspace is nil")
	}

	lim := Limiter{
		options: options,
		expire:  map[string]time.Time{},
	}

	cacheSizeBytes := options.GroupcacheSizeBytes
	if cacheSizeBytes == 0 {
		cacheSizeBytes = DefaultGroupcacheSizeBytes
	}

	cacheName := options.GroupcacheName
	if cacheName == "" {
		cacheName = "rate-limit"
	}

	getter := groupcache.GetterFunc(
		func(ctx context.Context, key string, dest groupcache.Sink) error {

			count := counter{
				Expire: time.Now().Add(options.Interval),
			}

			data, errMarshal := json.Marshal(count)
			if errMarshal != nil {
				return errMarshal
			}

			return dest.SetBytes(data, count.Expire)
		})

	lim.group = groupcache.NewGroupWithWorkspace(options.GroupcacheWorkspace, cacheName, cacheSizeBytes, getter)

	return &lim
}

type counter struct {
	Value  int       `json:"value"`
	Expire time.Time `json:"expire"`
}

// Consume attempts to consume the rate limiter.
// It returns true if the rate limiter allows access,
// or false if the rate limiter denies access.
func (l *Limiter) Consume(ctx context.Context, key string) (bool, error) {

	var dst []byte

	if errGet := l.group.Get(ctx, key,
		groupcache.AllocatingByteSliceSink(&dst)); errGet != nil {
		return true, errGet
	}

	var count counter
	errUnmarshal := json.Unmarshal(dst, &count)
	if errUnmarshal != nil {
		return true, errUnmarshal
	}

	count.Value++

	//
	// save key back with updated value
	//

	// 1/2: remove key
	if errRemove := l.group.Remove(ctx, key); errRemove != nil {
		log.Printf("ratelimit.Consume: remove key='%s' error: %v",
			key, errRemove)
	}

	remain := time.Until(count.Expire)
	expired := remain < 1

	// 2/2: reinsert key

	data, errMarshal := json.Marshal(count)
	if errMarshal != nil {
		return true, errMarshal
	}

	const hotCache = false // ???
	if !expired {
		if errSet := l.group.Set(ctx, key, data, count.Expire,
			hotCache); errSet != nil {
			return true, errSet
		}
	}

	accept := !expired && count.Value <= l.options.Slots

	log.Printf("ratelimit.Consume: key='%s' count=%d/%d interval=%v/%v expired=%t accept=%t",
		key, count.Value, l.options.Slots, remain, l.options.Interval, expired, accept)

	return accept, nil
}

/*
MetricsExporter creates a metrics exporter for Prometheus.

Usage example

	exporter := limiter.MetricsExporter()
	labels := map[string]string{
		"app": "app1",
	}
	namespace := ""
	collector := groupcache_exporter.NewExporter(namespace, labels, exporter)
	prometheus.MustRegister(collector)
	go func() {
		http.Handle(metricsRoute, promhttp.Handler())
		log.Fatal(http.ListenAndServe(metricsPort, nil))
	}()
*/
func (l *Limiter) MetricsExporter() *modernprogram.Group {
	return modernprogram.New(l.group)
}
