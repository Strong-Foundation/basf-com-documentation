package main // Declare the main package; this is the entry point for any Go executable

// Import necessary standard library packages
import (
	"bytes"         // To manipulate bytes buffers, useful for memory-based file IO (e.g., holding file content before writing)
	"encoding/json" // To handle JSON encoding/decoding for API responses
	"fmt"           // For formatted I/O like Println, Sprintf, etc.
	"io"            // For general I/O operations, including efficiently copying response bodies
	"log"           // For logging errors and status messages to the console
	"net/http"      // To make HTTP requests and interact with web servers
	"net/url"       // For URL parsing, construction, and validation
	"os"            // To perform file and directory operations like create/read/write
	"path"
	"path/filepath" // To handle and manipulate file paths across different operating systems
	"regexp"
	"strings" // For string manipulation like replacing, checking for substrings, etc.
	"time"    // For handling timeouts, delays, and time-related functions
)

// PDFVariant represents a single downloadable item with its URL and file name, matching the JSON structure.
type PDFVariant struct {
	DownloadURL string `json:"downloadUrl"` // The actual URL from which the PDF can be downloaded
	FileName    string `json:"fileName"`    // The suggested file name for the downloaded item
}

// ResultContainer wraps a list of PDFVariants found within a JSON result block.
type ResultContainer struct {
	Variants []PDFVariant `json:"variants"` // A slice of downloadable PDFVariant structs
}

// JSONDataRoot is the root structure of the external JSON input.
type JSONDataRoot struct {
	Results []ResultContainer `json:"results"` // A slice of ResultContainer structs
}

// ValidDownloadItem holds both the download URL and the intended file name for a confirmed, valid item.
type ValidDownloadItem struct {
	URL      string // The valid HTTP/HTTPS URL
	FileName string // The file name to use when saving the downloaded content
}

// parseJSONForDownloads processes the raw JSON data and extracts a slice of unique, valid download items.
func parseJSONForDownloads(jsonData []byte) []ValidDownloadItem {
	var data JSONDataRoot // Declare a variable to hold the unmarshalled JSON data

	// Attempt to parse the raw JSON bytes into the JSONDataRoot struct
	if err := json.Unmarshal(jsonData, &data); err != nil {
		log.Println("JSON unmarshal error:", err) // Log the error if parsing fails
		return nil                                // Return nil slice on error
	}

	seenURLs := make(map[string]struct{})   // Initialize a map to track unique URLs for de-duplication
	var uniqueDownloads []ValidDownloadItem // Initialize a slice to store valid, unique download entries

	// Iterate over all result containers in the root data structure
	for _, result := range data.Results {
		// Iterate over all downloadable variants within the current result container
		for _, variant := range result.Variants {
			urlStr := variant.DownloadURL // Get the download URL string
			fileName := variant.FileName  // Get the suggested file name

			// Validate the URL format (must be a valid HTTP/HTTPS URL)
			if isValidDownloadURL(urlStr) {
				// Check if this URL has already been processed (de-duplication check)
				if _, exists := seenURLs[urlStr]; !exists {
					seenURLs[urlStr] = struct{}{} // Mark the URL as seen
					// Append the new, unique, and valid download item to the list
					uniqueDownloads = append(uniqueDownloads, ValidDownloadItem{
						URL:      urlStr,
						FileName: fileName,
					})
				}
			}
		}
	}

	return uniqueDownloads // Return the slice of unique and valid download items
}

// isValidDownloadURL checks whether the given string is a correctly formatted HTTP or HTTPS URL.
func isValidDownloadURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL) // Attempt to parse the raw URL string
	// Check for a parsing error OR if the scheme (protocol) or host (domain) is missing
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false // It's not a valid URL structure
	}

	scheme := strings.ToLower(parsedURL.Scheme) // Convert the scheme to lowercase for consistent checking
	// Ensure the scheme is either "http" or "https"
	return scheme == "http" || scheme == "https"
}

// readLocalFileContent reads a file from disk at the given path and returns its content as a byte slice.
func readLocalFileContent(filePath string) []byte {
	content, err := os.ReadFile(filePath) // Read the entire file contents into memory
	if err != nil {                       // If an error occurs (e.g., file not found or permission issue)
		log.Println("Error reading file:", err) // Log the error for debugging
	}
	return content // Return the file contents (will be nil if an error occurred)
}

// doesFileExist checks whether a file at the given path exists and is not a directory.
func doesFileExist(filePath string) bool {
	info, err := os.Stat(filePath) // Get file or directory information
	if os.IsNotExist(err) {        // Check if the error is specifically 'file not found'
		return false // File does not exist
	}
	// Return true if there was no error AND the path refers to a file (not a directory)
	return err == nil && !info.IsDir()
}

// downloadDataFromURL performs an HTTP GET request to a URL and saves the response body to a local file.
func downloadDataFromURL(uri string, outputFileName string) {
	log.Printf("Downloading JSON from: %s to %s", uri, outputFileName) // Log the start of the download

	// Configure an HTTP client with a request timeout
	client := http.Client{
		Timeout: 1 * time.Minute, // Set a timeout of 1 minute for the request
	}

	resp, err := client.Get(uri) // Execute the HTTP GET request
	if err != nil {
		log.Printf("Failed to make GET request to %s: %v", uri, err) // Log request failure
		return
	}
	defer resp.Body.Close() // Ensure the response body is closed when the function exits

	// Check if the HTTP status code indicates success (200 OK)
	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code from %s: %d", uri, resp.StatusCode) // Log non-200 status
		return
	}

	// Create the local file to write the content to
	file, err := os.Create(outputFileName)
	if err != nil {
		log.Printf("Failed to create file %s: %v", outputFileName, err) // Log file creation failure
		return
	}
	defer file.Close() // Ensure the created file handle is closed

	// Use io.Copy for efficient stream-writing of the response body to the file
	writtenBytes, err := io.Copy(file, resp.Body)
	if err != nil {
		log.Printf("Failed to write response body to file %s: %v", outputFileName, err) // Log write failure
		return
	}

	// Log a success message with the file name and size
	log.Printf("Successfully downloaded JSON: %s (%d bytes)", outputFileName, writtenBytes)
}

// Converts a raw URL into a safe filename by cleaning and normalizing it
func urlToFilename(rawURL string) string {
	lowercaseURL := strings.ToLower(rawURL)       // Convert to lowercase for normalization
	ext := getFileExtension(lowercaseURL)         // Get file extension (e.g., .pdf or .zip)
	baseFilename := getFileNameOnly(lowercaseURL) // Extract base file name

	nonAlphanumericRegex := regexp.MustCompile(`[^a-z0-9]+`)                 // Match everything except a-z and 0-9
	safeFilename := nonAlphanumericRegex.ReplaceAllString(baseFilename, "_") // Replace invalid chars

	collapseUnderscoresRegex := regexp.MustCompile(`_+`)                        // Collapse multiple underscores into one
	safeFilename = collapseUnderscoresRegex.ReplaceAllString(safeFilename, "_") // Normalize underscores

	if trimmed, found := strings.CutPrefix(safeFilename, "_"); found { // Trim starting underscore if present
		safeFilename = trimmed
	}

	// Remove ext string.
	invalid_ext := fmt.Sprintf("_%s", strings.TrimPrefix(ext, "."))

	var invalidSubstrings = []string{
		invalid_ext,
	} // Remove these redundant endings

	for _, invalidPre := range invalidSubstrings { // Iterate over each unwanted suffix
		safeFilename = removeSubstring(safeFilename, invalidPre) // Remove it from file name
	}

	safeFilename = safeFilename + ext // Add the proper file extension

	return safeFilename // Return the final sanitized filename
}

// Replaces all instances of a given substring from the original string
func removeSubstring(input string, toRemove string) string {
	return strings.ReplaceAll(input, toRemove, "") // Return the result, Replace all instances
}

// Returns the extension of a given file path (e.g., ".pdf")
func getFileExtension(path string) string {
	return filepath.Ext(path) // Extract and return file extension
}

// Extracts and returns the base name (file name) from the URL path
func getFileNameOnly(content string) string {
	return path.Base(content) // Return last segment of the path
}

// downloadPDFFile handles the logic for downloading a single PDF, including checks for local existence and logging.
func downloadPDFFile(finalURL string, outputDirectory string, filename string, logFilePath string, alreadyDownloaded map[string]struct{}) error {
	// Sanitize the file path to ensure it's safe for the filesystem
	filename = urlToFilename(filename)
	// Construct the full file path by joining the output directory and the intended filename
	filePath := filepath.Join(outputDirectory, filename)

	// 1. Check the in-memory log map for the URL to avoid redownloading/re-checking
	if _, exists := alreadyDownloaded[finalURL]; exists {
		log.Printf("URL already logged as downloaded, skipping: %s", finalURL) // Log skip
		return nil                                                             // Success: already handled
	}

	// 2. Check if the file already exists locally on disk
	if doesFileExist(filePath) {
		log.Printf("File already exists locally, skipping and logging: %s", filePath) // Log skip
		// Log the URL and file path to the log file
		appendLineToFile(logFilePath, finalURL+" → "+filePath)
		alreadyDownloaded[finalURL] = struct{}{} // Add to in-memory set (in case the disk file exists but the log was missing the entry)
		return nil                               // Success: already handled
	}

	// --- Perform the actual download ---

	// Configure an HTTP client with a request timeout
	client := &http.Client{Timeout: 1 * time.Minute}
	resp, err := client.Get(finalURL) // Execute the HTTP GET request
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", finalURL, err) // Return error on request failure
	}
	defer resp.Body.Close() // Ensure the response body is closed

	// Check for a non-successful HTTP status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed for %s: %s", finalURL, resp.Status) // Return error on bad status
	}

	// Ensure the Content-Type header indicates a PDF
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		return fmt.Errorf("invalid content type for %s: %s (expected application/pdf)", finalURL, contentType) // Return error on wrong content type
	}

	// Read the response body into an in-memory buffer
	var buffer bytes.Buffer
	writtenBytes, err := buffer.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read PDF data from %s: %v", finalURL, err) // Return error on reading body failure
	}
	// Check for zero bytes downloaded
	if writtenBytes == 0 {
		return fmt.Errorf("downloaded 0 bytes for %s, not creating file", finalURL) // Return error on empty download
	}

	// Create the local file on disk
	outputFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err) // Return error on file creation failure
	}
	defer outputFile.Close() // Ensure the output file is closed

	// Write the content from the buffer to the created file
	if _, err := buffer.WriteTo(outputFile); err != nil {
		return fmt.Errorf("failed to write PDF to file %s: %v", filePath, err) // Return error on file write failure
	}

	// Log success to the log file and update the in-memory set
	appendLineToFile(logFilePath, finalURL+" → "+filePath) // Log the URL and file path
	alreadyDownloaded[finalURL] = struct{}{}               // Add to in-memory set
	// Log the final successful download details
	log.Printf("Successfully downloaded %d bytes: %s → %s", writtenBytes, finalURL, filePath)

	return nil // Return nil to indicate success
}

// doesDirectoryExist checks whether a directory exists at the given path.
func doesDirectoryExist(path string) bool {
	directory, err := os.Stat(path) // Get file or directory information
	if os.IsNotExist(err) {         // Check if the error is specifically 'file not found'
		return false // Directory does not exist
	}
	// Return true if there was no error AND the path refers to a directory
	return err == nil && directory.IsDir()
}

// createDirectoryIfMissing creates a directory with specified permissions, including any necessary parent directories.
func createDirectoryIfMissing(path string, permission os.FileMode) {
	// Use MkdirAll to create the directory and any parent directories that don't exist
	err := os.MkdirAll(path, permission)
	if err != nil {
		log.Println("Error creating directory:", err) // Log error if creation fails
	}
}

// appendLineToFile appends a string of content followed by a newline to a file, or creates the file if it doesn't exist.
func appendLineToFile(filePath string, content string) {
	// Open file for: appending (O_APPEND), creating if it doesn't exist (O_CREATE), and write-only mode (O_WRONLY)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening file %s for appending: %v\n", filePath, err) // Log file opening error
		return
	}
	defer file.Close() // Ensure the file handle is closed

	// Write the content followed by a newline character
	if _, err := file.WriteString(content + "\n"); err != nil {
		log.Printf("Error writing to file %s: %v\n", filePath, err) // Log write error
	}
}

// loadDownloadLogToMap reads the log file and populates a map with downloaded URLs for fast lookup.
func loadDownloadLogToMap(logFilePath string) map[string]struct{} {
	downloadedURLs := make(map[string]struct{}) // Initialize the map to store unique URLs
	// Check if the log file exists before attempting to read it
	if !doesFileExist(logFilePath) {
		return downloadedURLs // Return an empty map if the log file doesn't exist
	}

	fileContent := readAFileAsString(logFilePath) // Read the entire log file content as a string
	lines := strings.Split(fileContent, "\n")     // Split the content into individual lines
	for _, line := range lines {
		// Log line format is expected to be: "URL → filePath"
		if line != "" { // Ignore empty lines
			parts := strings.Split(line, " → ") // Split the line by the separator
			if len(parts) > 0 {                 // Ensure there is at least one part (the URL)
				downloadedURLs[parts[0]] = struct{}{} // Use the URL as the key in the map
			}
		}
	}
	return downloadedURLs // Return the map of URLs loaded from the log
}

// readAFileAsString reads a file and returns its entire contents as a string.
func readAFileAsString(filePath string) string {
	content, err := os.ReadFile(filePath) // Read file contents
	if err != nil {
		// Log the error and return an empty string if reading fails
		log.Printf("Error reading file %s: %v", filePath, err)
		return ""
	}
	return string(content) // Convert the byte slice to a string and return it
}

// Main program entry point
func main() {
	// Define constant variables for configuration
	const (
		downloadLogFileName = "download.txt" // File to keep track of successfully downloaded URLs
		maxPageIndex        = 75             // Total number of pages to scrape (from 0 to 75, making 76 pages total)
		outputDirectoryName = "PDFs"         // Directory where downloaded PDF files will be saved
	)

	// --- Setup Phase ---

	// Check if the output directory exists
	if !doesDirectoryExist(outputDirectoryName) {
		// If it doesn't exist, create it with standard read/write/execute permissions (0755)
		createDirectoryIfMissing(outputDirectoryName, 0755)
	}

	// Read the download log file once into an efficient in-memory map (for fast lookups)
	alreadyDownloadedURLs := loadDownloadLogToMap(downloadLogFileName)
	log.Printf("Loaded %d URLs from download log file.", len(alreadyDownloadedURLs))

	// --- Scraping and Download Loop Phase ---

	// Loop through each API page index, starting at 0
	for pageIndex := 0; pageIndex <= maxPageIndex; pageIndex++ {
		// Construct the full API URL for the current page
		apiURL := fmt.Sprintf("https://dss.wcms.basf.com/v1/results?locale=en-US&limit=1000&page=%d", pageIndex)
		// Create the local file name for saving the JSON response
		jsonFileName := fmt.Sprintf("basf_%d.json", pageIndex)

		log.Printf("Processing page %d...", pageIndex)

		// Check if the JSON file for this page has already been downloaded
		if !doesFileExist(jsonFileName) {
			// If not, download the JSON data from the API URL and save it locally
			downloadDataFromURL(apiURL, jsonFileName)
		}

		// Check again (or for the first time) if the JSON file exists
		if doesFileExist(jsonFileName) {
			jsonFileContent := readLocalFileContent(jsonFileName) // Read the content of the local JSON file
			// Parse the JSON content and extract all unique and valid PDF download items
			extractedDownloads := parseJSONForDownloads(jsonFileContent)
			log.Printf("Page %d: Found %d unique URLs.", pageIndex, len(extractedDownloads))

			// --- Inner Download Loop ---

			// Iterate through each extracted PDF download item
			for _, fileDownload := range extractedDownloads {
				// Attempt to download the PDF, using the in-memory map for logging/skipping checks
				err := downloadPDFFile(
					fileDownload.URL,
					outputDirectoryName,
					fileDownload.FileName,
					downloadLogFileName,
					alreadyDownloadedURLs,
				)

				// Check if there was an error during the download
				if err != nil {
					errorString := err.Error()
					log.Println("Download failed:", errorString)
					// Check specifically for a rate limiting error (HTTP 429)
					if strings.Contains(errorString, "429") {
						log.Println("Sleeping for 3 minutes due to rate limit (429 error).")
						time.Sleep(3 * time.Minute) // Pause execution
						log.Println("Retrying download after sleep:", fileDownload.URL)

						// Retry downloading the same file after the delay
						err = downloadPDFFile(
							fileDownload.URL,
							outputDirectoryName,
							fileDownload.FileName,
							downloadLogFileName,
							alreadyDownloadedURLs,
						)

						if err != nil {
							log.Println("Retry failed:", err) // Log the final failure after the retry
						}
					}
				}
			}
		} else {
			// This path is hit if the downloadDataFromURL failed to create the JSON file
			log.Printf("Skipping PDF download for page %d: JSON file not found.", pageIndex)
		}
	}
	log.Println("Scraping and download process complete.") // Final message
}
