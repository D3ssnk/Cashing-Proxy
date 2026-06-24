package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type CacheEntry struct {
	Header     http.Header
	Body       []byte
	StatusCode int
}

func main() {
	var mu sync.RWMutex
	cache := map[string]CacheEntry{}
	portPtr := flag.Int("port", 3000, "The port on which the caching proxy server will run.")
	originPtr := flag.String("origin", "", "The URL of the server to which the requests will be forwarded.")
	clearCachePtr := flag.Bool("clear-cache", false, "Clears the cache file.")
	flag.Parse()

	origin := *originPtr
	port := ":" + fmt.Sprint(*portPtr)
	clearCache := *clearCachePtr

	if clearCache {
		if _, err := os.Stat("cache.json"); !errors.Is(err, os.ErrNotExist) {
			err = os.Remove("cache.json")
			if err != nil {
				fmt.Println(err)
			}
		}
		return
	}

	if err := loadCache(&cache); err != nil {
		fmt.Println(err)
		return
	}

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
			header = val.Header
			statusCode = val.StatusCode
			body = val.Body

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

			cacheEntry.Header = header
			cacheEntry.Body = body
			cacheEntry.StatusCode = statusCode

			mu.Lock()

			cache[fullURL] = cacheEntry
			if err := writeToCache(cache); err != nil {
				mu.Unlock()
				fmt.Println(err)
				return
			}
			mu.Unlock()
		}

		maps.Copy(w.Header(), header)
		if inCache {
			w.Header().Add("X-Cache", "HIT")
		} else {
			w.Header().Add("X-Cache", "MISS")
		}
		w.WriteHeader(statusCode)
		w.Write(body)
	})

	http.ListenAndServe(port, nil)
}

func writeToCache(cache map[string]CacheEntry) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return errors.New("Error parsing as JSON")
	}
	err = os.WriteFile("cache.json", data, 0644)

	if err != nil {
		return errors.New("Error writing JSON to the file")
	}

	return nil
}

func loadCache(cache *map[string]CacheEntry) error {
	if _, err := os.Stat("cache.json"); errors.Is(err, os.ErrNotExist) {
		writeToCache(*cache)
		return nil
	}
	body, err := os.ReadFile("cache.json")
	if err != nil {
		return errors.New("Error reading file")
	}
	err = json.Unmarshal(body, cache)
	if err != nil {
		return errors.New("Error parsing as JSON")
	}
	return nil
}
