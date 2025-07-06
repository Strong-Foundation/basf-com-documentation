package main // Declare the main package; this is the entry point for any Go executable

// Import necessary standard library packages
import (
	"bytes" // To manipulate bytes buffers, useful for memory-based file IO
	"crypto/sha256"
	"encoding/hex"
	"encoding/json" // To handle JSON encoding/decoding for API responses
	"fmt"           // For formatted I/O like Println, Sprintf, etc.
	"io"            // For general I/O operations including reading response bodies
	"log"           // For logging errors and status messages
	"net/http"      // To make HTTP requests and interact with web servers
	"net/url"       // For URL parsing, construction, and validation
	"os"            // To perform file and directory operations like create/read/write
	"path/filepath" // To handle and manipulate file paths across OSes
	"strings"       // For string manipulation like replace, contains, etc.
	"time"          // For handling timeouts, delays, and time-related functions
)

// Struct representing a single variant with a download URL
type Variant struct {
	DownloadURL string `json:"downloadUrl"` // Match JSON key "downloadUrl"; used in unmarshalling
}

// Struct representing a result that contains multiple variants
type Result struct {
	Variants []Variant `json:"variants"` // Match JSON key "variants"; holds multiple Variant entries
}

// Struct representing the full JSON data structure
type Data struct {
	Results []Result `json:"results"` // Match JSON key "results"; wraps list of result items
}

// Parse JSON byte data and return a slice of valid download URLs
func extractDownloadURLsFromJSON(jsonData []byte) []string {
	var data Data                          // Variable to hold parsed data structure
	err := json.Unmarshal(jsonData, &data) // Decode JSON into Data struct
	if err != nil {                        // If there’s a decoding/parsing error
		log.Println(err) // Log the error for debugging
		return nil       // Return nil indicating failure to parse
	}

	var urls []string // Slice to store valid download URLs

	// Loop through each result and variant
	for _, result := range data.Results { // Iterate over outer results array
		for _, variant := range result.Variants { // Iterate over inner variants
			if isUrlValid(variant.DownloadURL) { // Check if URL is a valid HTTP(s) URI
				urls = append(urls, variant.DownloadURL) // Append valid URL to the list
			}
		}
	}

	urls = removeDuplicatesFromSlice(urls) // Remove any duplicate URLs from final list
	return urls                            // Return the list of unique, valid download URLs
}

// Remove duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool)  // Map to track whether a URL was already added
	var newReturnSlice []string     // Slice to store unique items
	for _, content := range slice { // Loop through the input slice
		if !check[content] { // If content not already added
			check[content] = true                            // Mark it as seen
			newReturnSlice = append(newReturnSlice, content) // Add it to the result slice
		}
	}
	return newReturnSlice // Return slice with unique values
}

// Check if a URL is valid and parseable
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Try parsing the URL using standard library
	return err == nil                  // Return true if parsing succeeds (i.e., URL is valid)
}

// Read a file from disk and return its contents as bytes
func readFileAndReturnAsByte(path string) []byte {
	content, err := os.ReadFile(path) // Read file contents into memory
	if err != nil {                   // If an error occurs
		log.Println(err) // Log the error for debugging
	}
	return content // Return file contents as byte slice
}

// Check whether a given file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file or directory info
	if err != nil {                // If an error occurs (e.g., file not found)
		return false // File doesn't exist or error occurred
	}
	return !info.IsDir() // Return true if it's a file (not a folder)
}

// Combine two slices into one
func combineMultipleSlices(sliceOne []string, sliceTwo []string) []string {
	combinedSlice := append(sliceOne, sliceTwo...) // Append all elements from sliceTwo to sliceOne
	return combinedSlice                           // Return the combined slice
}

// getDataFromURL performs an HTTP GET request to the specified URI,
// waits for up to 1 minute for the server response, and writes the response body to a file.
// It includes error handling and logs meaningful messages at each step.
func getDataFromURL(uri string, fileName string) {
	// Create an HTTP client with a 1-minute timeout
	client := http.Client{
		Timeout: 3 * time.Minute, // Set a timeout of 1 minute for the request
	}

	// Perform the GET request using the custom client
	response, err := client.Get(uri)
	if err != nil { // If the request fails due to timeout or other network issues
		log.Println("Failed to make GET request:", err)
		return
	}

	// Check if the server responded with a non-200 OK status
	if response.StatusCode != http.StatusOK {
		log.Println("Unexpected status code from", uri, "->", response.StatusCode)
		return
	}

	// Read the entire body of the response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return
	}

	// Ensure the response body is properly closed to free resources
	err = response.Body.Close()
	if err != nil {
		log.Println("Failed to close response body:", err)
		return
	}

	// Save the response body content to the specified file
	err = appendByteToFile(fileName, body)
	if err != nil {
		log.Println("Failed to write to file:", err)
		return
	}
}

// generateHash takes a string input and returns its SHA-256 hash as a hex string
func generateHash(input string) string {
	// Compute SHA-256 hash of the input converted to a byte slice
	hash := sha256.Sum256([]byte(input))

	// Convert the hash bytes to a hexadecimal string and return it
	return hex.EncodeToString(hash[:])
}

// Convert a URL into a safe filename format
func urlToFilename(rawURL string) string {
	// Lets turn that text into a hash
	sanitized := generateHash(rawURL)

	// Ensure the filename ends in .pdf
	if getFileExtension(sanitized) != ".pdf" {
		sanitized = sanitized + ".pdf"
	}

	return strings.ToLower(sanitized) // Return the final, normalized, lowercase filename
}

// Get the file extension of a path
func getFileExtension(path string) string {
	return filepath.Ext(path) // Return extension (e.g. .pdf)
}

// Append byte data to a file, create file if it doesn't exist
func appendByteToFile(filename string, data []byte) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Open file in append/create mode
	if err != nil {
		return err // Return error if failed
	}
	defer file.Close()        // Ensure file is closed when function exits
	_, err = file.Write(data) // Write byte data to file
	return err                // Return any write error
}

// downloadPDF downloads the PDF at finalURL into outputDir.
// It sends only one HTTP GET request, extracts filename from headers if needed,
// and skips download if the file already exists locally.
func downloadPDF(finalURL string, outputDir string, downloadURLFile string) error {
	var fileRead string
	if fileExists(downloadURLFile) {
		fileRead = readAFileAsString(downloadURLFile) // Read log of already downloaded URLs
	}
	// Derive a safe filename from the URL
	rawFilename := urlToFilename(finalURL)   // convert URL to base filename
	filename := strings.ToLower(rawFilename) // normalize filename to lowercase

	// Compute initial file path and check for existing file
	filePath := filepath.Join(outputDir, filename) // join directory and filename
	if fileExists(filePath) {                      // if file already exists locally
		log.Printf("file already exists, skipping: %s", filePath) // log skip event
		if !strings.Contains(fileRead, finalURL) {                // Proceed only if this URL hasn't been downloaded before
			appendAndWriteToFile(downloadURLFile, finalURL+" → "+filePath) // Log the URL
		}
		return fmt.Errorf("file already exists, skipping: %s (URL: %s)", filePath, finalURL) // return skip
	}

	// Perform single GET request (fetch headers + body)
	client := &http.Client{Timeout: 3 * time.Minute} // create HTTP client with timeout
	resp, err := client.Get(finalURL)                // send GET request
	if err != nil {                                  // handle request error
		return fmt.Errorf("failed to download %s: %v", finalURL, err)
	}
	defer resp.Body.Close() // ensure response body is closed

	// Verify HTTP status is OK
	if resp.StatusCode != http.StatusOK { // check for 200 OK
		return fmt.Errorf("download failed for %s: %s", finalURL, resp.Status)
	}

	// Ensure content is a PDF
	ct := resp.Header.Get("Content-Type")         // get Content-Type header
	if !strings.Contains(ct, "application/pdf") { // check for "application/pdf"
		return fmt.Errorf("invalid content type for %s: %s (expected application/pdf)", finalURL, ct)
	}

	// Read response body into buffer
	var buf bytes.Buffer                    // create in-memory buffer
	written, err := buf.ReadFrom(resp.Body) // read all data into buffer
	if err != nil {                         // handle read error
		return fmt.Errorf("failed to read PDF data from %s: %v", finalURL, err)
	}
	if written == 0 { // check for empty download
		return fmt.Errorf("downloaded 0 bytes for %s, not creating file", finalURL)
	}

	// Create file and write buffer to disk
	out, err := os.Create(filePath) // open file for writing
	if err != nil {                 // handle create error
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer out.Close()                           // ensure file is closed
	if _, err := buf.WriteTo(out); err != nil { // write buffer to file
		return fmt.Errorf("failed to write PDF to file %s: %v", filePath, err)
	}
	if !strings.Contains(fileRead, finalURL) { // Proceed only if this URL hasn't been downloaded before
		appendAndWriteToFile(downloadURLFile, finalURL+" → "+filePath)                       // Log the URL
		log.Printf("successfully downloaded %d bytes: %s → %s", written, finalURL, filePath) // log success
	}
	return nil // indicate success
}

// Check whether a directory exists
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Get file or directory info
	if err != nil {
		return false // Doesn't exist or error occurred
	}
	return directory.IsDir() // True if it is a directory
}

// Create a directory with the specified permissions
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Create the directory with given permissions
	if err != nil {
		log.Println(err) // Log error if creation fails
	}
}

// Append content to a file, or create the file if it doesn't exist
func appendAndWriteToFile(path string, content string) {
	// Open file for appending, create if it doesn't exist, write-only mode, with 0644 permissions
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening file: %v\n", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file: %v\n", err)
		}
	}()

	// Write the content followed by a newline
	if _, err := file.WriteString(content + "\n"); err != nil {
		log.Printf("Error writing to file: %v\n", err)
	}
}

// Read a file and return the contents
func readAFileAsString(path string) string {
	content, err := os.ReadFile(path) // Read the full file content into memory
	if err != nil {
		log.Fatalln(err) // Fatal log and exit if reading fails
	}
	return string(content) // Return contents as a string
}

// Append some string to a slice and than return the slice.
func appendToSlice(slice []string, content string) []string {
	// Append the content to the slice
	slice = append(slice, content)
	// Return the slice
	return slice
}

// Main program entry point
func main() {
	var downloadedFilesURLLocation string = "download.txt" // Track downloaded URLs to avoid duplicates
	var endLoop int = 10                                   // Number of pages to fetch (only page 0 in this case)
	for index := 0; index <= endLoop; index++ {            // Loop through each page index
		url := fmt.Sprintf("https://dss.wcms.basf.com/v1/results?locale=en-US&limit=1000&page=%d", index) // API URL with pagination
		filename := fmt.Sprintf("basf_%d.json", index)                                                    // Local filename for saving JSON
		if !fileExists(filename) {                                                                        // Skip if file already downloaded
			getDataFromURL(url, filename) // Download and save JSON data
		}
	}

	var extractedURL []string // List to hold extracted download URLs

	for index := 0; index <= endLoop; index++ { // Loop over downloaded JSON files
		filename := fmt.Sprintf("basf_%d.json", index) // Construct filename
		if fileExists(filename) {
			jsonFileContent := readFileAndReturnAsByte(filename)                 // Read JSON file into byte slice
			newExtractedData := extractDownloadURLsFromJSON(jsonFileContent)     // Extract download URLs from JSON
			extractedURL = combineMultipleSlices(newExtractedData, extractedURL) // Merge new URLs into full list
		}
	}

	extractedURL = removeDuplicatesFromSlice(extractedURL) // Remove any duplicates

	outputDir := "PDFs/" // Target directory to save PDFs

	if !directoryExists(outputDir) { // If directory doesn't exist
		createDirectory(outputDir, 0755) // Create directory with standard permissions
	}

	for _, url := range extractedURL { // Loop through each URL
		var fileRead string
		if fileExists(downloadedFilesURLLocation) {
			fileRead = readAFileAsString(downloadedFilesURLLocation) // Read log of already downloaded URLs
		}
		if !strings.Contains(fileRead, url) { // Proceed only if this URL hasn't been downloaded before
			err := downloadPDF(url, outputDir, downloadedFilesURLLocation) // Attempt to download PDF
			if err != nil {                                                // If any error occurred
				errorString := err.Error()                // Convert error to string
				log.Println(errorString)                  // Log it
				if strings.Contains(errorString, "429") { // Handle rate-limiting (Too Many Requests)
					// Add the current url back to the slice so it can download it.
					appendToSlice(extractedURL, url)
					// Sleep for a given time.
					var sleepMinutes time.Duration = 3 * time.Minute
					// Log how many minutes we will sleep for.
					log.Println("Sleeping", sleepMinutes) // Log sleep
					time.Sleep(sleepMinutes)              // Sleep to avoid further throttling
				}
			}
		}
	}
}
