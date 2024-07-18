package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
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
	const specificText = "https://choiceqr.com"

	// Create a rate limiter: ticker that ticks every 200ms with a random delay
	ticker := time.NewTicker(50*time.Millisecond + time.Duration(rand.Intn(250))*time.Millisecond)
	defer ticker.Stop()

	// Output file to write positive results
	outputFile, err := os.Create("positive_results.txt")
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	checkedCount := 0

	// Check each domain one by one
	for _, domain := range domains {
		<-ticker.C

		httpUrl := "http://" + domain.Domain
		httpsUrl := "https://" + domain.Domain

		found, err := checkDomainForText(httpUrl, specificText)
		// if err != nil && strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
		// 	fmt.Println("Pausing for 5 minutes due to timeout error")
		// 	time.Sleep(5 * time.Minute)
		// 	ticker = time.NewTicker(200*time.Millisecond + time.Duration(rand.Intn(250))*time.Millisecond)
		// 	<-ticker.C
		if err == nil && found {
			fmt.Printf("Found '%s' on %s (HTTP)\n", specificText, domain.Domain)
			_, err := outputFile.WriteString(fmt.Sprintf("%s (HTTP)\n", domain.Domain))
			if err != nil {
				fmt.Println("Error writing to output file:", err)
			}
			checkedCount++
			continue
		}

		found, err = checkDomainForText(httpsUrl, specificText)
		// if err != nil && strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
		// 	fmt.Println("Pausing for 5 minutes due to timeout error")
		// 	time.Sleep(5 * time.Minute)
		// 	ticker = time.NewTicker(200*time.Millisecond + time.Duration(rand.Intn(250))*time.Millisecond)
		// 	<-ticker.C
		// } else {
		if found {
			fmt.Printf("Found '%s' on %s (HTTPS)\n", specificText, domain.Domain)
			_, err := outputFile.WriteString(fmt.Sprintf("%s (HTTPS)\n", domain.Domain))
			if err != nil {
				fmt.Println("Error writing to output file:", err)
			}
		} else {
			fmt.Printf("'%s' not found on %s\n", specificText, domain.Domain)
		}
		checkedCount++
		// }
		fmt.Printf("Checked %d domains\n", checkedCount)
	}
}

func checkDomainForText(url, text string) (bool, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
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
