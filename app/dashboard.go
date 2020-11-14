package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

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

	// Define this way so that the handler holds the client
	handler := func(response http.ResponseWriter, _ *http.Request) {

		log.Print("Fetching Ingresses.")
		services := GetApps(client)

		log.Print("Rendering")
		// Sub in the services to the table
		tmplt.Execute(response, services)
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
