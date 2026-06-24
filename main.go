package main

import (
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type CacheEntry struct {
	header http.Header
	body []byte
	statusCode int 
}

func main() {
	var mu sync.RWMutex
	cache := map[string]CacheEntry{}
	portPtr := flag.Int("port", 3000, "The port on which the caching proxy server will run.")
	originPtr := flag.String("origin", "", "The URL of the server to which the requests will be forwarded.")
	flag.Parse()

	origin := *originPtr
	port := ":" + fmt.Sprint(*portPtr)

	

	if parsedURL, err := url.Parse(origin); err != nil || parsedURL.Host == "" ||
		(!strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://")) {
		panic("Bad!")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		
		var header http.Header
		var statusCode int
		var body []byte
		fullURL := origin + r.URL.RequestURI()

		mu.RLock()
		val, inCache := cache[fullURL]
		mu.RUnlock()

		if inCache {
			header = val.header
			statusCode = val.statusCode
			body = val.body
			
		} else {
			resp, err := http.Get(fullURL)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprintf(w, "%s", err.Error())
				return
			}
			defer resp.Body.Close()

			resBody, err := io.ReadAll(resp.Body)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprintf(w, "%s", err.Error())
				return
			}
			header = resp.Header
			statusCode = resp.StatusCode
			body = resBody

			var cacheEntry CacheEntry

			cacheEntry.header = header
			cacheEntry.body = body
			cacheEntry.statusCode = statusCode

			mu.Lock()
			cache[fullURL] = cacheEntry
			mu.Unlock()	
		}
		
		maps.Copy(w.Header(), header)
		if inCache {w.Header().Add("X-Cache", "HIT")} else {w.Header().Add("X-Cache", "MISS")}
		w.WriteHeader(statusCode)
		w.Write(body)
	})

	http.ListenAndServe(port, nil)
}
