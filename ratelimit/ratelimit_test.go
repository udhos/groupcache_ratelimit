package ratelimit

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/modernprogram/groupcache/v2"
)

func TestRateLimit(t *testing.T) {

	//
	// create servers
	//

	const numServers = 10
	const interval = 200 * time.Millisecond
	const slots = 10
	var servers []*testServer
	var peers []string

	for i := 0; i < numServers; i++ {

		workspace := groupcache.NewWorkspace()

		addr := fmt.Sprintf("127.0.0.1:%d", 8000+i)

		myURL := "http://" + addr

		peers = append(peers, myURL)

		pool := groupcache.NewHTTPPoolOptsWithWorkspace(workspace, myURL, &groupcache.HTTPPoolOptions{})

		t.Logf("server %d: workspace=%p pool=%p", i, workspace, pool)

		limOptions := Options{
			Interval:            interval,
			Slots:               slots,
			GroupcacheWorkspace: workspace,
			Debug:               true,
		}

		s := &testServer{
			workspace: workspace,
			pool:      pool,
			server: &http.Server{
				Handler: pool,
				Addr:    addr,
			},
			limiter: New(limOptions),
		}

		servers = append(servers, s)
	}

	t.Logf("peers: %v", peers)

	for _, s := range servers {
		s.pool.Set(peers...)
	}

	//
	// start servers
	//

	for _, s := range servers {
		go func() {
			log.Printf("groupcache server: listening on %s", s.server.Addr)
			err := s.server.ListenAndServe()
			log.Printf("groupcache server %s: exited: %v", s.server.Addr, err)
		}()
	}

	//
	// Run tests
	//

	const wait = 200 * time.Millisecond
	t.Logf("giving time for http servers to start: %v", wait)
	time.Sleep(wait)

	lim := servers[0].limiter

	send(t, lim, "key1", slots, true)
	send(t, lim, "key1", 50, false)
	send(t, lim, "key2", slots, true)
	send(t, lim, "key2", 50, false)

	t.Logf("replenishing slots: %v", interval)
	time.Sleep(interval)

	send(t, lim, "key1", slots, true)
	send(t, lim, "key1", 50, false)
	send(t, lim, "key2", slots, true)
	send(t, lim, "key2", 50, false)

	//
	// shutdown servers
	//

	for _, s := range servers {
		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if errShut := s.server.Shutdown(ctx); errShut != nil {
			t.Logf("http shutdown error: %v", errShut)
		}
	}
}

func send(t *testing.T, lim *Limiter, key string, n int, expectAccept bool) {
	for i := 1; i <= n; i++ {
		accept, errLim := lim.Consume(context.TODO(), key)
		if errLim != nil {
			t.Errorf("Consume error: %v", errLim)
		}
		if expectAccept != accept {
			t.Errorf("%d/%d: accept: key=%s expected=%v got=%v",
				i, n, key, expectAccept, accept)
		}
	}
}

type testServer struct {
	workspace *groupcache.Workspace
	pool      *groupcache.HTTPPool
	server    *http.Server
	limiter   *Limiter
}
