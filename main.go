package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// Route defines a simple routing rule
type Route struct {
	Host    string `json:"host"`
	Path    string `json:"path"`
	Backend string `json:"backend"`
}

// Config holds the routing rules
type Config struct {
	Port   string  `json:"port"`
	Routes []Route `json:"routes"`
}

var config Config

// loadConfig reads the routing configuration from a file
func loadConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &config); err != nil {
		return err
	}
	return nil
}

// findBackend finds the backend URL for the given request
func findBackend(r *http.Request) string {
	for _, route := range config.Routes {
		if r.Host == route.Host && strings.HasPrefix(r.URL.Path, route.Path) {
			return route.Backend
		}
	}
	return ""
}

// reverseProxy forwards the request to the selected backend
func reverseProxy(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Invalid backend URL: %s", target)
	}

	return httputil.NewSingleHostReverseProxy(targetURL)
}

func handler(w http.ResponseWriter, r *http.Request) {
	backend := findBackend(r)
	if backend == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	log.Printf("Routing %s%s -> %s", r.Host, r.URL.Path, backend)
	reverseProxy(backend).ServeHTTP(w, r)
}

func main() {
	configFile := "config.json"
	if err := loadConfig(configFile); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	port := ":" + config.Port
	if config.Port == "" {
		port = ":8080"
	}

	log.Printf("Starting ingress controller on %s", port)
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
