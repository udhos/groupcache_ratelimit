package main

import (
	"log"

	"github.com/modernprogram/groupcache/v2"
)

func startGroupcache() *groupcache.Workspace {

	workspace := groupcache.NewWorkspace()

	//
	// create groupcache pool
	//

	groupcachePort := ":5000"

	myURL := "http://127.0.0.1" + groupcachePort

	log.Printf("groupcache my URL: %s", myURL)

	pool := groupcache.NewHTTPPoolOptsWithWorkspace(workspace, myURL, &groupcache.HTTPPoolOptions{})

	/*
		//
		// start groupcache server
		//

		serverGroupCache := &http.Server{Addr: groupcachePort, Handler: pool}

		go func() {
			log.Printf("groupcache server: listening on %s", groupcachePort)
			err := serverGroupCache.ListenAndServe()
			log.Printf("groupcache server: exited: %v", err)
		}()
	*/

	pool.Set(myURL)

	return workspace
}
