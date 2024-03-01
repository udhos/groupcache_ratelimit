// Package main implements the example tool.
package main

import (
	"context"
	"log"
	"time"

	"github.com/udhos/groupcache_ratelimit/ratelimit"
)

func main() {

	var lim *ratelimit.Limiter

	interval := 1 * time.Second

	lim = newLimiter(interval, 5)
	send(lim, 5)

	lim = newLimiter(interval, 5)
	send(lim, 10)
	send(lim, 10)

	sleep(interval)
	send(lim, 10)
}

func sleep(d time.Duration) {
	log.Printf("sleeping %v", d)
	time.Sleep(d)
}

func newLimiter(interval time.Duration, slots int) *ratelimit.Limiter {

	groupcacheWorkspace := startGroupcache()

	log.Printf("interval=%v slots=%d", interval, slots)

	options := ratelimit.Options{
		Interval:            interval,
		Slots:               slots,
		GroupcacheWorkspace: groupcacheWorkspace,
	}

	lim := ratelimit.New(options)

	metrics(lim)

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
