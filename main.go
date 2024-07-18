package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type DomainInfo struct {
	Domain      string
	RegistrarID string
	HolderID    string
	NS          string
	ExpiryDate  string
}

type Result struct {
	Found   bool
	Err     error
	Domain  DomainInfo
	Protocol string // This field will store either "HTTP" or "HTTPS"
}

func main() {
	// Open the file
	file, err := os.Open("domains.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	var domains []DomainInfo

	// Read and parse the file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ";")
		if len(parts) != 5 {
			fmt.Println("Invalid line:", line)
			continue
		}
		domains = append(domains, DomainInfo{
			Domain:      parts[0],
			RegistrarID: parts[1],
			HolderID:    parts[2],
			NS:          parts[3],
			ExpiryDate:  parts[4],
		})
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Specific text to find
	const specificText = "google"

	// Create a wait group and a channel to collect results
	var wg sync.WaitGroup
	results := make(chan Result, len(domains))
	var checkedCount int32

	// Rate limiter: ticker that ticks every 200ms with a random delay
	ticker := time.NewTicker(200*time.Millisecond + time.Duration(rand.Intn(250))*time.Millisecond)
	defer ticker.Stop()

	// Channel to signal a pause
	pause := make(chan struct{}, 1)

	// Start a goroutine for each domain check
	for _, domain := range domains {
		wg.Add(1)
		go func(domain DomainInfo) {
			defer wg.Done()

			for {
				select {
				case <-ticker.C:
					httpUrl := "http://" + domain.Domain
					httpsUrl := "https://" + domain.Domain

					found, err := checkDomainForText(httpUrl, specificText)
					if err != nil && strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
						pause <- struct{}{}
					} else if err == nil && found {
						results <- Result{Found: true, Err: nil, Domain: domain, Protocol: "HTTP"}
						atomic.AddInt32(&checkedCount, 1)
						return
					} else {
						found, err := checkDomainForText(httpsUrl, specificText)
						if err != nil && strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
							pause <- struct{}{}
						} else {
							results <- Result{Found: found, Err: err, Domain: domain, Protocol: "HTTPS"}
							atomic.AddInt32(&checkedCount, 1)
							return
						}
					}

				case <-pause:
					fmt.Println("Pausing for 5 minutes due to timeout error")
					time.Sleep(5 * time.Minute)
					ticker = time.NewTicker(200*time.Millisecond + time.Duration(rand.Intn(250))*time.Millisecond)
				}
			}
		}(domain)
	}

	// Close the results channel once all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and print results, write positives to a file
	outputFile, err := os.Create("positive_results.txt")
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	for result := range results {
		if result.Err != nil {
			fmt.Printf("Error fetching domain %s: %v\n", result.Domain.Domain, result.Err)
		} else if result.Found {
			fmt.Printf("Found '%s' on %s (%s)\n", specificText, result.Domain.Domain, result.Protocol)
			_, err := outputFile.WriteString(fmt.Sprintf("%s (%s)\n", result.Domain.Domain, result.Protocol))
			if err != nil {
				fmt.Println("Error writing to output file:", err)
			}
		} else {
			fmt.Printf("'%s' not found on %s\n", specificText, result.Domain.Domain)
		}
	}

	fmt.Printf("Checked %d domains\n", atomic.LoadInt32(&checkedCount))
}

func checkDomainForText(url, text string) (bool, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	// Change the User-Agent to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(body), text), nil
}
