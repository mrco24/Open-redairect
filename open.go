package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const (
	redColor   = "\033[31m"
	resetColor = "\033[0m"
)

func main() {
	urlListFile := flag.String("l", "", "File containing a list of URLs")
	singleURL := flag.String("u", "", "Single URL to test")
	payloadFile := flag.String("p", "", "File containing payloads to append")
	outputFile := flag.String("o", "", "Output file to save results")
	threads := flag.Int("t", 1, "Number of threads (goroutines) to use")
	verbose := flag.Bool("v", false, "Print verbose output")
	flag.Parse()

	if *urlListFile == "" && *singleURL == "" {
		fmt.Println("Please provide either a URL list file (-l) or a single URL (-u)")
		return
	}

	if *payloadFile == "" {
		fmt.Println("Please provide a payload file (-p)")
		return
	}

	// Create a channel to communicate VULNERABLE results to the file writer
	vulnerableResults := make(chan string)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, *threads)

	// Start the file writer goroutine
	go writeResultsToFile(vulnerableResults, *outputFile)

	// Custom HTTP client to handle redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Stop redirects after the first one
			return http.ErrUseLastResponse
		},
	}

	for _, originalURL := range getURLs(*urlListFile, *singleURL) {
		// Trim any trailing slashes to avoid double slashes
		originalURL = strings.TrimSuffix(originalURL, "/")

		payloads, err := ioutil.ReadFile(*payloadFile)
		if err != nil {
			fmt.Println("Error reading payload file:", err)
			return
		}

		for _, payload := range strings.Split(string(payloads), "\n") {
			payload := strings.TrimSpace(payload) // Trim any leading/trailing spaces

			// Construct the test URL without an extra slash
			testURL := fmt.Sprintf("%s%s", originalURL, payload)

			wg.Add(1)
			semaphore <- struct{}{}

			go func(testURL string) {
				defer func() {
					<-semaphore
					wg.Done()
				}()

				resp, err := client.Get(testURL)
				if err != nil {
					fmt.Printf("%s -> Error: %s\n", testURL, err)
					return
				}

				if resp.StatusCode >= 300 && resp.StatusCode < 400 {
					// Print the actual redirection target
					redirectedURL := resp.Header.Get("Location")

					// Parse the original and redirected URLs
					originalParsedURL, err := url.Parse(originalURL)
					if err != nil {
						fmt.Printf("Error parsing original URL: %s\n", err)
						return
					}

					redirectedParsedURL, err := url.Parse(redirectedURL)
					if err != nil {
						fmt.Printf("Error parsing redirected URL: %s\n", err)
						return
					}

					// Check if the hostnames are different, indicating subdomain redirection
					if originalParsedURL.Hostname() != redirectedParsedURL.Hostname() {
						result := fmt.Sprintf("%s -> %sVULNERABLE%s (Redirected to %s)\n", testURL, redColor, resetColor, redirectedURL)
						fmt.Print(result)

						// Send VULNERABLE result to the file writer
						vulnerableResults <- result
					} else if *verbose {
						// Print non-vulnerable results if in verbose mode
						result := fmt.Sprintf("%s -> Not vulnerable (Redirected to %s)\n", testURL, redirectedURL)
						fmt.Print(result)
					}
				} else {
					if *verbose {
						// Print non-vulnerable results if in verbose mode
						result := fmt.Sprintf("%s -> Not vulnerable\n", testURL)
						fmt.Print(result)
					}
				}
				resp.Body.Close()
			}(testURL)
		}
	}

	wg.Wait()

	// Close the channel to signal the file writer to finish
	close(vulnerableResults)
}

// writeResultsToFile is a goroutine that writes VULNERABLE results to the output file.
func writeResultsToFile(results <-chan string, outputFile string) {
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer file.Close()

	for result := range results {
		_, err := file.WriteString(result)
		if err != nil {
			fmt.Println("Error writing to output file:", err)
			return
		}
	}
}

// getURLs retrieves the list of URLs based on command-line flags.
func getURLs(urlListFile, singleURL string) []string {
	if urlListFile != "" {
		fileURLs, err := ioutil.ReadFile(urlListFile)
		if err != nil {
			fmt.Println("Error reading URL list file:", err)
			return nil
		}
		return strings.Split(string(fileURLs), "\n")
	} else if singleURL != "" {
		return []string{singleURL}
	}
	return nil
}
