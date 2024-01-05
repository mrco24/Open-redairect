package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	urlListFile := flag.String("l", "", "File containing a list of URLs")
	singleURL := flag.String("u", "", "Single URL to test")
	payloadFile := flag.String("p", "", "File containing payloads to append")
	flag.Parse()

	if *urlListFile == "" && *singleURL == "" {
		fmt.Println("Please provide either a URL list file (-l) or a single URL (-u)")
		return
	}

	if *payloadFile == "" {
		fmt.Println("Please provide a payload file (-p)")
		return
	}

	payloads, err := ioutil.ReadFile(*payloadFile)
	if err != nil {
		fmt.Println("Error reading payload file:", err)
		return
	}

	var urls []string
	if *urlListFile != "" {
		fileURLs, err := ioutil.ReadFile(*urlListFile)
		if err != nil {
			fmt.Println("Error reading URL list file:", err)
			return
		}
		urls = strings.Split(string(fileURLs), "\n")
	} else {
		urls = []string{*singleURL}
	}

	// Custom HTTP client to handle redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Stop redirects after the first one
			return http.ErrUseLastResponse
		},
	}

	for _, originalURL := range urls {
		// Trim any trailing slashes to avoid double slashes
		originalURL = strings.TrimSuffix(originalURL, "/")

		for _, payload := range strings.Split(string(payloads), "\n") {
			payload = strings.TrimSpace(payload) // Trim any leading/trailing spaces

			// Construct the test URL without an extra slash
			testURL := fmt.Sprintf("%s%s", originalURL, payload)

			resp, err := client.Get(testURL)
			if err != nil {
				fmt.Printf("%s -> Error: %s\n", testURL, err)
				continue
			}

			if resp.StatusCode >= 300 && resp.StatusCode < 400 {
				// Print the actual redirection target
				redirectedURL := resp.Header.Get("Location")

				// Parse the original and redirected URLs
				originalParsedURL, err := url.Parse(originalURL)
				if err != nil {
					fmt.Printf("Error parsing original URL: %s\n", err)
					continue
				}

				redirectedParsedURL, err := url.Parse(redirectedURL)
				if err != nil {
					fmt.Printf("Error parsing redirected URL: %s\n", err)
					continue
				}

				// Check if the hostnames are different, indicating subdomain redirection
				if originalParsedURL.Hostname() != redirectedParsedURL.Hostname() {
					fmt.Printf("%s -> VULNERABLE (Redirected to %s)\n", testURL, redirectedURL)
				} else {
					fmt.Printf("%s -> Not vulnerable (Redirected to %s)\n", testURL, redirectedURL)
				}
			} else {
				fmt.Printf("%s -> Not vulnerable\n", testURL)
			}
			resp.Body.Close()
		}
	}
}
