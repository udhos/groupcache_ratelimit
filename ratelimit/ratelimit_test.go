package ratelimit

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
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

	for i := range numServers {

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
	// Run concurrent tests
	//

	var stats testStats
	var wg sync.WaitGroup
	const numClients = 2
	const period = 1 * time.Second
	const delay = 20 * time.Millisecond
	release := make(chan struct{})

	wg.Add(numClients)

	for i := range numClients {
		go func() {
			t.Logf("client %d: waiting", i)
			<-release

			begin := time.Now()

			for time.Since(begin) < period {
				accept, errLim := lim.Consume(context.TODO(), "key3")

				switch {
				case errLim != nil:
					t.Errorf("client %d: ERROR: Consume: %v", i, errLim)
					stats.error()
				case accept:
					stats.accept()
				default:
					stats.reject()
				}

				time.Sleep(delay)
			}

			wg.Done()
			t.Logf("client %d: done", i)
		}()
	}

	close(release)

	wg.Wait()

	t.Errorf("clients=%d period=%v delay=%v accepts=%d rejects=%d errors=%d",
		numClients, period, delay, stats.accepts, stats.rejects, stats.errors)

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

type testStats struct {
	lock    sync.Mutex
	accepts int
	rejects int
	errors  int
}

func (s *testStats) accept() {
	s.lock.Lock()
	s.accepts++
	s.lock.Unlock()
}

func (s *testStats) reject() {
	s.lock.Lock()
	s.rejects++
	s.lock.Unlock()
}

func (s *testStats) error() {
	s.lock.Lock()
	s.errors++
	s.lock.Unlock()
}
