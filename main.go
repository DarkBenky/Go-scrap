package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	Found    bool
	Err      error
	Domain   DomainInfo
	Protocol string // This field will store either "HTTP" or "HTTPS"
}

func main() {
	// Track the start time
	startTime := time.Now()

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
	const specificText = "choiceqr"

	// Open output file for positive results
	outputFile, err := os.Create("positive_results.txt")
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	// Statistics
	var checkedCount int
	totalDomains := len(domains)

	// Scan each site sequentially
	for _, domain := range domains {
		domainStartTime := time.Now() // Start time for each domain

		httpUrl := "http://" + domain.Domain
		httpsUrl := "https://" + domain.Domain

		// Check HTTP
		found, err := checkDomainForText(httpUrl, specificText)
		if err != nil {
			fmt.Printf("Error fetching domain %s: %v\n", domain.Domain, err)
		} else if found {
			fmt.Printf("Found '%s' on %s (HTTP)\n", specificText, domain.Domain)
			_, err := outputFile.WriteString(fmt.Sprintf("%s (HTTP)\n", domain.Domain))
			if err != nil {
				fmt.Println("Error writing to output file:", err)
			}
			checkedCount++
			logProgress(checkedCount, totalDomains, domainStartTime)
			continue
		}

		// Check HTTPS
		found, err = checkDomainForText(httpsUrl, specificText)
		if err != nil {
			fmt.Printf("Error fetching domain %s: %v\n", domain.Domain, err)
		} else if found {
			fmt.Printf("Found '%s' on %s (HTTPS)\n", specificText, domain.Domain)
			_, err := outputFile.WriteString(fmt.Sprintf("%s (HTTPS)\n", domain.Domain))
			if err != nil {
				fmt.Println("Error writing to output file:", err)
			}
		}

		checkedCount++
		logProgress(checkedCount, totalDomains, domainStartTime)
	}

	// Log final statistics
	totalTime := time.Since(startTime)
	fmt.Printf("Scan complete. Checked %d domains in %s.\n", checkedCount, totalTime)
}

// logProgress logs the progress of the scan, including the time taken for each domain.
func logProgress(checkedCount, totalDomains int, startTime time.Time) {
	elapsed := time.Since(startTime)
	fmt.Printf("Checked %d/%d domains. Time taken for this domain: %s. Estimated time remaining: %s.\n",
		checkedCount, totalDomains, elapsed, estimateRemainingTime(checkedCount, totalDomains, elapsed))
}

// estimateRemainingTime estimates the remaining time based on the average time per domain.
func estimateRemainingTime(checkedCount, totalDomains int, timePerDomain time.Duration) time.Duration {
	if checkedCount == 0 {
		return 0
	}
	averageTime := timePerDomain / time.Duration(checkedCount)
	remainingDomains := totalDomains - checkedCount
	return averageTime * time.Duration(remainingDomains)
}

func checkDomainForText(url, text string) (bool, error) {
	client := &http.Client{
		Timeout: 1 * time.Second, // Increased timeout
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
