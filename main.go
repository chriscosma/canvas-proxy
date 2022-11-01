package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
)

const CANVAS_DOMAIN_PATTERN string = "(?m)^[a-z]+\\.instructure\\.com$"

func reqHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("[DEBUG]", "GET params were: ", req.URL.Query())

	canvas_domain := req.URL.Query().Get("canvas_domain")

	// Check if canvas_domain key is present
	if canvas_domain == "" {
		// No canvas_domain key was present
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "missing canvas_domain query parameter")
		return
	}

	// Check format of canvas_domain key
	matched, err := regexp.MatchString(CANVAS_DOMAIN_PATTERN, canvas_domain)
	if err != nil {
		// Something went wrong while parsing domain string
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("[ERROR]", err)
		io.WriteString(w, "something went wrong while parsing the Canvas domain")
		return
	}
	if !matched {
		// Invalid canvas_domain key
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "canvas_domain has bad format")
		return
	}

	// Check if access token key is present
	access_token := req.URL.Query().Get("access_token")
	if access_token == "" {
		// No access_token key was present
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "missing access_token query parameter")
		return
	}

	// Form request to send to Canvas REST API
	request_domain := fmt.Sprint("https://", canvas_domain, "/api/v1", req.URL.Path)
	canvas_req, err := http.NewRequest("GET", request_domain, nil)
	if err != nil {
		log.Println("[ERROR]", err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "something went wrong while forming the request to send to Canvas")
		return
	}

	// Add authorization header
	canvas_req.Header.Add("Authorization", fmt.Sprint("Bearer ", access_token))

	// Add user-passed query parameters to send to Canvas excluding access_token and canvas_domain
	canvas_query := req.URL.Query()
	for key := range req.URL.Query() {
		if key != "access_token" && key != "canvas_domain" {
			canvas_query.Add(key, req.URL.Query().Get(key))
		}
	}
	canvas_req.URL.RawQuery = canvas_query.Encode()

	// Send request
	client := &http.Client{}
	canvas_resp, err := client.Do(canvas_req)
	if err != nil {
		log.Println("[ERROR]", err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "something went wrong while sending the request to send to Canvas")
		return
	}
	defer canvas_resp.Body.Close()

	// Read and send back response from Canvas
	canvas_resp_body, err := io.ReadAll(canvas_resp.Body)
	if err != nil {
		log.Println("[ERROR]", "something went wrong while reading the Canvas response body", canvas_resp_body)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "something went wrong while reading the Canvas response body")
		return
	}
	// Add link header if it exists
	link_header := canvas_resp.Header.Get("Link")
	if link_header != "" {
		w.Header().Set("Link", link_header)
	}
	// Set content type if it's going to be JSON
	if canvas_resp.StatusCode == 200 {
		w.Header().Set("Content-Type", "application/json")
	}

	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "app://obsidian.md")
	w.Header().Set("Access-Control-Expose-Headers", "Link")

	// Send response to user
	w.WriteHeader(canvas_resp.StatusCode)
	w.Write(canvas_resp_body)
}

func main() {
	http.HandleFunc("/", reqHandler)

	log.Println("[INFO] listening for requests at http://localhost:8080/")

	log.Fatal("[FATAL]", http.ListenAndServe(":8080", nil))
}
