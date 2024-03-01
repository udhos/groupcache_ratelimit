// Package ratelimit implements rate limiting with groupcache.
package ratelimit

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
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

	RemoveBeforeReinsert bool
	ReinsertExpired      bool
}

// Limiter implements rate limiting.
type Limiter struct {
	options Options
	group   *groupcache.Group
	expire  map[string]time.Time
	lock    sync.Mutex
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

			expire := time.Now().Add(options.Interval)

			lim.setExpire(key, expire) // save expire

			return dest.SetBytes([]byte("0"), expire)
		})

	lim.group = groupcache.NewGroupWithWorkspace(options.GroupcacheWorkspace, cacheName, cacheSizeBytes, getter)

	return &lim
}

func (l *Limiter) getExpire(key string) (time.Time, bool) {
	l.lock.Lock()
	e, found := l.expire[key]
	l.lock.Unlock()
	return e, found
}

func (l *Limiter) setExpire(key string, expire time.Time) {
	l.lock.Lock()
	l.expire[key] = expire
	l.lock.Unlock()
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

	str := string(dst)

	counter, errConv := strconv.Atoi(str)
	if errConv != nil {
		return true, errConv
	}

	counter++
	data := []byte(strconv.Itoa(counter))

	expire, hasExpire := l.getExpire(key) // keep key existing expire
	if !hasExpire {
		panic(fmt.Sprintf("key expire not set: key='%s'", key))
	}
	if expire.IsZero() {
		panic(fmt.Sprintf("key expire is zero: key='%s'", key))
	}

	//
	// save key back with updated value
	//

	// 1/2: remove key
	if l.options.RemoveBeforeReinsert {
		if errRemove := l.group.Remove(ctx, key); errRemove != nil {
			log.Printf("ratelimit.Consume: remove key='%s' error: %v",
				key, errRemove)
		}
	}

	remain := time.Until(expire)
	expired := remain > 0

	// 2/2: reinsert key
	const hotCache = false // ???
	if l.options.ReinsertExpired {
		if errSet := l.group.Set(ctx, key, data, expire,
			hotCache); errSet != nil {
			return true, errSet
		}
	} else {
		if !expired {
			if errSet := l.group.Set(ctx, key, data, expire,
				hotCache); errSet != nil {
				return true, errSet
			}
		}
	}

	accept := !expired && counter <= l.options.Slots

	log.Printf("ratelimit.Consume: key='%s' count=%d/%d interval=%v/%v expired=%t accept=%t",
		key, counter, l.options.Slots, remain, l.options.Interval, expired, accept)

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
