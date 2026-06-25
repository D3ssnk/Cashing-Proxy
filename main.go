// Package main implements a simple HTTP caching proxy server.
// It forwards requests to a specified origin server, caches the responses in memory
// and on disk (cache.json), and serves subsequent identical requests from the cache.
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

// CacheEntry represents a single cached HTTP response.
// It stores the necessary components to perfectly reconstruct the response for the client.
type CacheEntry struct {
    Header     http.Header
    Body       []byte
    StatusCode int
}

func main() {
    // mu provides concurrent-safe access to the in-memory cache map.
    // RWMutex allows multiple simultaneous readers, but locks out everyone during a write.
    var mu sync.RWMutex
    cache := map[string]CacheEntry{}

    // Define command-line flags for configuring the proxy.
    portPtr := flag.Int("port", 3000, "The port on which the caching proxy server will run.")
    originPtr := flag.String("origin", "", "The URL of the server to which the requests will be forwarded.")
    clearCachePtr := flag.Bool("clear-cache", false, "Clears the cache file.")
    flag.Parse()

    origin := *originPtr
    port := ":" + fmt.Sprint(*portPtr)
    clearCache := *clearCachePtr

    // Handle the --clear-cache flag.
    // If set, attempt to delete the cache.json file and exit the program.
    if clearCache {
        if _, err := os.Stat("cache.json"); !errors.Is(err, os.ErrNotExist) {
            err = os.Remove("cache.json")
            if err != nil {
                fmt.Println(err)
            }
        }
        return
    }

    // Initialize the in-memory cache by loading data from the disk.
    if err := loadCache(&cache); err != nil {
        fmt.Println(err)
        return
    }

    // Validate the provided origin URL to ensure it is properly formatted 
    // and uses a valid HTTP/HTTPS scheme before starting the server.
    if parsedURL, err := url.Parse(origin); err != nil || parsedURL.Host == "" ||
        (!strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://")) {
        panic("Bad!")
    }

    // Register the proxy handler function to intercept all incoming HTTP requests.
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

        var header http.Header
        var statusCode int
        var body []byte
        
        // Reconstruct the full destination URL by appending the requested URI to the origin.
        fullURL := origin + r.URL.RequestURI()

        // Acquire a read lock to safely check if the requested URL is already cached.
        mu.RLock()
        val, inCache := cache[fullURL]
        mu.RUnlock()

        if inCache {
            // CACHE HIT: Retrieve the response details from memory.
            header = val.Header
            statusCode = val.StatusCode
            body = val.Body

        } else {
            // CACHE MISS: Forward the request to the origin server.
            resp, err := http.Get(fullURL)
            if err != nil {
                w.WriteHeader(500)
                fmt.Fprintf(w, "%s", err.Error())
                return
            }
            defer resp.Body.Close()

            // Read the entire response body into memory.
            resBody, err := io.ReadAll(resp.Body)
            if err != nil {
                w.WriteHeader(500)
                fmt.Fprintf(w, "%s", err.Error())
                return
            }
            
            // Capture the response details.
            header = resp.Header
            statusCode = resp.StatusCode
            body = resBody

            var cacheEntry CacheEntry
            cacheEntry.Header = header
            cacheEntry.Body = body
            cacheEntry.StatusCode = statusCode

            // Acquire a write lock to safely update the in-memory map and disk file.
            mu.Lock()

            cache[fullURL] = cacheEntry
            
            // Persist the updated cache state to the disk.
            if err := writeToCache(cache); err != nil {
                mu.Unlock()
                fmt.Println(err)
                return
            }
            mu.Unlock()
        }

        // Copy the captured headers (either from cache or origin) to the client response.
        maps.Copy(w.Header(), header)
        
        // Inject custom X-Cache header to indicate if the response was served from cache.
        if inCache {
            w.Header().Add("X-Cache", "HIT")
        } else {
            w.Header().Add("X-Cache", "MISS")
        }
        
        // Write the final status code and body back to the client.
        w.WriteHeader(statusCode)
        w.Write(body)
    })

    // Start the HTTP server.
    http.ListenAndServe(port, nil)
}

// writeToCache serializes the in-memory cache map to JSON and writes it to cache.json.
func writeToCache(cache map[string]CacheEntry) error {
    data, err := json.Marshal(cache)
    if err != nil {
        return errors.New("Error parsing as JSON")
    }
    
    // Write the JSON data to disk with standard 0644 file permissions.
    err = os.WriteFile("cache.json", data, 0644)

    if err != nil {
        return errors.New("Error writing JSON to the file")
    }

    return nil
}

// loadCache reads cache.json from disk and deserializes it into the provided cache map.
// If the file does not exist, it creates a new, empty cache.json file.
func loadCache(cache *map[string]CacheEntry) error {
    if _, err := os.Stat("cache.json"); errors.Is(err, os.ErrNotExist) {
        // Create an empty cache file if it's missing to prevent future read errors.
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