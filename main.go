package main

import (
	"flag"
	"fmt"
	"net/http"
)

func main() {
    portPtr := flag.Int("port", 3000, "The port on which the caching proxy server will run.")
	originPtr := flag.String("origin","", "The URL of the server to which the requests will be forwarded.")

	flag.Parse()
	fmt.Println(*portPtr, *originPtr)
}
