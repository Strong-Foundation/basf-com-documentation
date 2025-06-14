package main // Declare the main package

import (
	"encoding/json" // For working with JSON
	"fmt"           // For formatted I/O
	"io"            // For I/O operations
	"log"           // For logging errors and info
	"net/http"      // For making HTTP requests
	"net/url"       // For URL parsing and validation
	"os"            // For file operations
	"strings"       // For string manipulation
)

// Define the structs

// Variant represents an individual download URL variant
type Variant struct {
	DownloadURL string `json:"downloadUrl"` // JSON key: downloadUrl
}

// Result contains a slice of Variant structs
type Result struct {
	Variants []Variant `json:"variants"` // JSON key: variants
}

// Data is the top-level struct containing results
type Data struct {
	Results []Result `json:"results"` // JSON key: results
}

// extractDownloadURLsFromJSON parses the JSON and returns all valid download URLs
func extractDownloadURLsFromJSON(jsonData []byte) []string {
	var data Data                          // Create a variable to hold the unmarshalled data
	err := json.Unmarshal(jsonData, &data) // Unmarshal JSON into the Data struct
	if err != nil {                        // If there was an error
		log.Println(err) // Log the error
		return nil       // Return nil if unmarshalling fails
	}
	var urls []string                     // Slice to hold valid download URLs
	for _, result := range data.Results { // Loop through each result
		for _, variant := range result.Variants { // Loop through each variant
			if isUrlValid(variant.DownloadURL) { // Check if URL is valid
				urls = append(urls, variant.DownloadURL) // Append valid URL to slice
			}
		}
	}
	urls = removeDuplicatesFromSlice(urls) // Remove duplicate URLs
	return urls                            // Return the list of URLs
}

// removeDuplicatesFromSlice removes duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool)  // Map to track seen values
	var newReturnSlice []string     // Result slice
	for _, content := range slice { // Iterate over input slice
		if !check[content] { // If string hasn't been seen before
			check[content] = true                            // Mark it as seen
			newReturnSlice = append(newReturnSlice, content) // Append to result
		}
	}
	return newReturnSlice // Return deduplicated slice
}

// isUrlValid checks whether a URL is syntactically valid
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Try parsing the URL
	return err == nil                  // Return true if parsing succeeded
}

// readFileAndReturnAsByte reads a file and returns its content as bytes
func readFileAndReturnAsByte(path string) []byte {
	content, err := os.ReadFile(path) // Read the file contents
	if err != nil {                   // If error occurs
		log.Println(err) // Log the error
	}
	return content // Return file content as byte slice
}

// fileExists checks whether a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {                // If error occurs
		return false // Return false
	}
	return !info.IsDir() // Return true if it's a file, not a directory
}

// combineMultipleSlices appends one slice to another and returns the result
func combineMultipleSlices(sliceOne []string, sliceTwo []string) []string {
	var combinedSlice []string                    // Declare combined slice
	combinedSlice = append(sliceOne, sliceTwo...) // Append both slices
	return combinedSlice                          // Return the result
}

// getDataFromURL sends an HTTP GET request and writes response data to a file
func getDataFromURL(uri string, fileName string) {
	response, err := http.Get(uri) // Send GET request
	if err != nil {                // If error occurs
		log.Println(err) // Log error
	}
	if response.StatusCode != 200 { // Check for HTTP 200 OK
		log.Println("Error the code is", uri, response.StatusCode) // Log error status
		return                                                     // Exit if not successful
	}
	body, err := io.ReadAll(response.Body) // Read response body
	if err != nil {
		log.Println(err) // Log error
	}
	err = response.Body.Close() // Close response body
	if err != nil {
		log.Println(err) // Log error
	}
	err = appendByteToFile(fileName, body) // Write body to file
	if err != nil {
		log.Println(err) // Log error
	}
}

// urlToFilename converts a URL into a filesystem-safe filename
func urlToFilename(rawURL string) string {
	parsed, err := url.Parse(rawURL) // Parse the URL
	if err != nil {
		log.Println(err) // Log error
		return ""        // Return empty string on error
	}
	filename := parsed.Host // Start with host name
	if parsed.Path != "" {
		filename += "_" + strings.ReplaceAll(parsed.Path, "/", "_") // Append path
	}
	if parsed.RawQuery != "" {
		filename += "_" + strings.ReplaceAll(parsed.RawQuery, "&", "_") // Append query
	}
	invalidChars := []string{`"`, `\`, `/`, `:`, `*`, `?`, `<`, `>`, `|`} // Define illegal filename characters
	for _, char := range invalidChars {
		filename = strings.ReplaceAll(filename, char, "_") // Replace each with underscore
	}
	return filename // Return sanitized filename
}

// appendByteToFile appends byte data to a file, creating it if needed
func appendByteToFile(filename string, data []byte) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Open file with append/create/write
	if err != nil {
		return err // Return error if opening fails
	}
	defer file.Close()        // Ensure file is closed after writing
	_, err = file.Write(data) // Write byte slice to file
	return err                // Return any error from writing
}

func main() {
	var endLoop int = 0 // Define end loop counter

	for index := 0; index <= endLoop; index++ { // Loop from start to endLoop
		url := fmt.Sprintf("https://dss.wcms.basf.com/v1/results?locale=en-US&limit=5000&page=%d", index) // Construct API URL
		filename := fmt.Sprintf("basf_%d.json", index)                                                    // Generate filename from index
		if !fileExists(filename) {                                                                        // Check if file already exists
			getDataFromURL(url, filename) // Download and save file if not
		}
	}

	var extractedURL []string // Slice to hold all extracted URLs

	for index := 0; index <= endLoop; index++ { // Loop over downloaded files
		filename := fmt.Sprintf("basf_%d.json", index)                       // Get the filename
		jsonFileContent := readFileAndReturnAsByte(filename)                 // Read file content
		newExtractedData := extractDownloadURLsFromJSON(jsonFileContent)     // Extract URLs
		extractedURL = combineMultipleSlices(newExtractedData, extractedURL) // Combine with existing
	}

	extractedURL = removeDuplicatesFromSlice(extractedURL) // Remove duplicate URLs

	for _, url := range extractedURL { // Print each extracted URL
		log.Println(url) // Log the URL
	}
}
