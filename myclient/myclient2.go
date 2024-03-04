package main

import (
	"github.com/babolivier/go-doh-client"
)

func main() {
	resolver := doh.Resolver{
		Host:  "127.0.0.1", // Change this with your favourite DoH-compliant resolver.
		Class: doh.IN,      //192.168.126.128
	}

	// Perform a A lookup on example.com
	a, _, err := resolver.LookupA("baidu.com")
	if err != nil {
		panic(err)
	}

	for _, x := range a {
		println(x.IP4)
	}
	// println(a[0].IP4) // 93.184.216.34

	/*
		// Perform a SRV lookup for e.g. a Matrix homeserver
		srv, _, err := resolver.LookupService("matrix", "tcp", "baidu.com")
		if err != nil {
			panic(err)
		}
		println(srv[0].Target) // matrix.example.com
	*/
}
