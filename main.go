package main

import (
	"fmt"
	"flag"
	//"net/http"
	"net/http"
	"net/url"
	"strings"
)

func main() {
    portPtr := flag.Int("port", 3000, "The port on which the caching proxy server will run.")
	originPtr := flag.String("origin","", "The URL of the server to which the requests will be forwarded.")
	flag.Parse()

	origin := *originPtr
	port := fmt.Sprint(*portPtr)
	if  parsedURL, err := url.Parse(origin); err != nil || parsedURL.Host == "" ||
	(!strings.HasPrefix(origin,"http://") && !strings.HasPrefix(origin,"https://"))  {
		panic("Bad!")
	}
	
	// http.HandleFunc("/", handler)

	http.ListenAndServe(port, nil)
}

// func handler (w http.ResponseWriter, r *http.Request) {
// 	origin 
// 	resp, err := http.Get()
// }