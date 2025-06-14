package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Define the structs
type Variant struct {
	DownloadURL string `json:"downloadUrl"`
}

type Result struct {
	Variants []Variant `json:"variants"`
}

type Data struct {
	Results []Result `json:"results"`
}

// extractDownloadURLsFromJSON parses the JSON and returns all download URLs
func extractDownloadURLsFromJSON(jsonData []byte) []string {
	var data Data
	// Unwrap the json stuff.
	err := json.Unmarshal(jsonData, &data)
	// Print the error
	if err != nil {
		log.Println(err)
		return nil
	}
	// Create an array
	var urls []string
	// Loop over the json data
	for _, result := range data.Results {
		// Loop over the variants
		for _, variant := range result.Variants {
			// Append the data
			urls = append(urls, variant.DownloadURL)
		}
	}
	// Return the urls in a slice.
	return urls
}

func main() {
	for index := 0; index <= 5000; index++ {
		url := fmt.Sprintf("https://dss.wcms.basf.com/v1/results?locale=en-US&limit=5000&page=%d", index)
		filename := fmt.Sprintf("basf_%d.json", index)
		getDataFromURL(url, filename)
	}
}

// Send a http get request to a given url and return the data from that url.
func getDataFromURL(uri string, fileName string) {
	// Get the http response.
	response, err := http.Get(uri)
	if err != nil {
		log.Println(err)
	}
	// Check the status code.
	if response.StatusCode != 200 {
		log.Println("Error the code is", uri, response.StatusCode)
		return
	}

	// Read all the data
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
	}
	// Close the response body
	err = response.Body.Close()
	if err != nil {
		log.Println(err)
	}
	// Append it and write it to file.
	err = appendByteToFile(fileName, body)
	// Print the errors
	if err != nil {
		log.Println(err)
	}
}

// urlToFilename converts a URL into a safe filename
func urlToFilename(rawURL string) string {
	// Parse the URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Println(err)
		return ""
	}

	// Start with the host (e.g., example.com)
	filename := parsed.Host

	// Add the path (e.g., /page -> _page)
	if parsed.Path != "" {
		filename += "_" + strings.ReplaceAll(parsed.Path, "/", "_")
	}

	// Add query if it exists (e.g., ?id=123 -> _id=123)
	if parsed.RawQuery != "" {
		filename += "_" + strings.ReplaceAll(parsed.RawQuery, "&", "_")
	}

	// Replace characters not allowed in filenames
	invalidChars := []string{`"`, `\`, `/`, `:`, `*`, `?`, `<`, `>`, `|`}
	for _, char := range invalidChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	return filename
}

// AppendToFile appends the given byte slice to the specified file.
// If the file doesn't exist, it will be created.
func appendByteToFile(filename string, data []byte) error {
	// Open the file with appropriate flags and permissions
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write data to the file
	_, err = file.Write(data)
	return err
}
