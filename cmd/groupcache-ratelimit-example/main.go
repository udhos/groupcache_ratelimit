// Package main implements the example tool.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/modernprogram/groupcache/v2"
	"github.com/udhos/groupcache_ratelimit/ratelimit"
)

func main() {

	var lim limiter

	interval := 1 * time.Second

	lim = newLimiter(interval, 5)
	send(lim.rateLimiter, 5)
	lim.stop()

	lim = newLimiter(interval, 5)
	send(lim.rateLimiter, 10)
	send(lim.rateLimiter, 10)

	sleep(interval)
	send(lim.rateLimiter, 10)
	lim.stop()
}

func sleep(d time.Duration) {
	log.Printf("sleeping %v", d)
	time.Sleep(d)
}

type limiter struct {
	rateLimiter *ratelimit.Limiter
	server      *http.Server
}

func (l *limiter) stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := l.server.Shutdown(ctx)
	if err != nil {
		log.Printf("http server shutdown: %v", err)
	}
}

func newLimiter(interval time.Duration, slots int) limiter {

	workspace := groupcache.NewWorkspace()

	log.Printf("interval=%v slots=%d", interval, slots)

	addr := "127.0.0.1:5000"
	myURL := "http://" + addr

	pool := groupcache.NewHTTPPoolOptsWithWorkspace(workspace, myURL, &groupcache.HTTPPoolOptions{})

	pool.Set(myURL)

	options := ratelimit.Options{
		Interval:            interval,
		Slots:               slots,
		GroupcacheWorkspace: workspace,
	}

	lim := limiter{
		rateLimiter: ratelimit.New(options),
		server: &http.Server{
			Handler: pool,
			Addr:    addr,
		},
	}

	go func() {
		log.Printf("groupcache server: listening on %s", addr)
		err := lim.server.ListenAndServe()
		log.Printf("groupcache server %s: exited: %v", addr, err)
	}()

	metrics(lim.rateLimiter)

	return lim
}

func send(lim *ratelimit.Limiter, amount int) {

	ctx := context.TODO()
	const key = "1.1.1.1"

	var accepts int
	var errors int
	for i := 1; i <= amount; i++ {
		acc, errCall := lim.Consume(ctx, key)
		if errCall != nil {
			errors++
			log.Printf("send %d/%d: errors=%d", i, amount, errors)
			continue
		}
		if acc {
			accepts++
		}
	}
	log.Printf("send: amount=%d accepts=%d rejects=%d errors=%d",
		amount, accepts, amount-accepts, errors)
}
