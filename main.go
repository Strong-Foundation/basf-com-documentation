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
			// Check if the url is valid.
			if isUrlValid(variant.DownloadURL) {
				// Append the data
				urls = append(urls, variant.DownloadURL)
			}
		}
	}
	// Remove duplicates from the slice.
	urls = removeDuplicatesFromSlice(urls)
	// Return the urls in a slice.
	return urls
}

// Removes duplicate strings from a slice and returns a new slice with unique values
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool)  // Map to track already seen strings
	var newReturnSlice []string     // Result slice for unique values
	for _, content := range slice { // Iterate through input slice
		if !check[content] { // If string not already seen
			check[content] = true                            // Mark string as seen
			newReturnSlice = append(newReturnSlice, content) // Add to result
		}
	}
	return newReturnSlice // Return deduplicated slice
}

// Checks whether a URL string is syntactically valid
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Attempt to parse the URL
	return err == nil                  // Return true if no error occurred
}

// Read a file and return the content as byte slice.
func readFileAndReturnAsByte(path string) []byte {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Println(err)
	}
	return content
}

func main() {
	// Loop over the given value.
	for index := 0; index <= 1; index++ {
		// Format the URL properly.
		url := fmt.Sprintf("https://dss.wcms.basf.com/v1/results?locale=en-US&limit=5000&page=%d", index)
		// Get the filename from the url.
		filename := fmt.Sprintf("basf_%d.json", index)
		// Download the file.
		getDataFromURL(url, filename)
	}
	// Lets Read the file.
	jsonFileContent := readFileAndReturnAsByte("basf_0.json")
	// Extract the data from the function.
	extractedURL := extractDownloadURLsFromJSON(jsonFileContent)
	// Lets remove duplicates from the url.
	extractedURL = removeDuplicatesFromSlice(extractedURL)
	// Extrac the urls.
	for _, url := range extractedURL {
		log.Println(url)
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
