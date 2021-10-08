package main

import (
	"math/rand"
	"net/http"
	"strings"
	"time"
)

/*
probably doesn't matter, but when somone is scanning our website,
send them 200 response with completely random data.
let them choke on it.
the urls come from observing logs
*/

var (
	badClients = map[string]bool{
		"/images/":                               true,
		"/files/":                                true,
		"/uploads/":                              true,
		"/admin/controller/extension/extension/": true,
		"/sites/default/files/":                  true,
		"/.well-known/":                          true,
	}
	badClientsContains = []string{
		"/wp-login.php",
		"/wp-includes/wlwmanifest.xml",
		"/xmlrpc.php",
		"/wp-admin",
		"/wp-content/",
	}
	badClientsRandomData []byte
)

func init() {
	d := make([]byte, 0, 1024)
	d = append(d, []byte("<html><body>fuck you...")...)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	start := len(d)
	for i := start; i < 1024; i++ {
		d = append(d, byte(rnd.Intn(256)))
	}
	badClientsRandomData = d
}

// returns true if sent a response to the client
func tryServeBadClient(w http.ResponseWriter, r *http.Request) bool {
	isBadClient := func(uri string) bool {
		if badClients[uri] {
			return true
		}
		for _, s := range badClientsContains {
			if strings.Contains(uri, s) {
				return true
			}
		}
		return false
	}
	if !isBadClient(r.URL.Path) {
		return false
	}
	w.Header().Add("Content-Tyep", "text/html")
	w.WriteHeader(200)
	w.Write(badClientsRandomData)
	return true
}
