package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

const cache_duration = "1m"

// Cache the app list
type AppCache struct {
	apps      []UserService
	timestamp time.Time
}

func main() {
	log.Print("Starting up.")

	// The kubernetes client
	client, _ := GetClient()

	// The template to render
	tmplt, err := template.ParseFiles("/srv/index.html")
	if err != nil {
		log.Print("Failed")
		panic(err.Error())
	}

	// Start with an ancient timestamp, which will be invalidated.
	cache := AppCache{make([]UserService, 0), time.Time{}}

	timeout, err := time.ParseDuration(cache_duration)
	if err != nil {
		log.Print("Invalid cache duration.")
		panic(err.Error())
	}

	// Define this way so that the handler holds the client
	handler := func(response http.ResponseWriter, _ *http.Request) {

		if time.Since(cache.timestamp) > timeout {
			log.Print("Fetching Ingresses.")
			cache = AppCache{GetApps(client), time.Now()}
		} else {
			log.Print("Using cache.")
		}

		log.Print("Rendering")
		// Sub in the services to the table
		tmplt.Execute(response, cache.apps)
	}

	// Configure PORT via env var
	port_str := os.Getenv("PORT")
	if len(port_str) == 0 {
		port_str = "8000"
	}
	port_str = fmt.Sprintf(":%s", port_str)

	// Run the webserver
	fs := http.FileServer(http.Dir("/srv/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(port_str, nil))
}
