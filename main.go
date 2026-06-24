package main

import (
	"flag"
	"fmt"
	"io"

	//"net/http"
	"net/http"
	"net/url"
	"strings"
)

func main() {
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
		fullURL := origin + r.URL.Path
		resp, err := http.Get(fullURL)
		if err != nil {
			w.WriteHeader(resp.StatusCode)
			fmt.Fprintf(w, "%s", err.Error())
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(resp.StatusCode)
			fmt.Fprintf(w, "%s", err.Error())
			return
		}

		w.Header().Add("X-Cache", "MISS")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})

	http.ListenAndServe(port, nil)
}
