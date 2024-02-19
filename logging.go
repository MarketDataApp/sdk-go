// Package client provides a collection of structs for logging.
// The Market Data Go SDK provides a [comprehensive logging framework] tailored for HTTP request and response tracking.
// It facilitates detailed logging of HTTP interactions, including request details, response data, error handling, and rate limit monitoring.
// The logging functionality is designed to support a wide range of market data types such as stocks, options, indices, and market status information,
// making it an essential tool for developers integrating market data into their applications.
//
// # Key Features
//
//   - Detailed Logging: Captures and logs comprehensive details of HTTP requests and responses, including headers, status codes, and body content.
//   - Error Handling: Distinguishes between client-side and server-side errors, logging them appropriately to aid in debugging and monitoring.
//   - Rate Limit Management: Tracks and logs rate limit consumption for each request, helping to avoid rate limit breaches.
//   - Debug Mode: Offers a debug mode that logs additional details for in-depth analysis during development and troubleshooting.
//   - In-Memory Log Storage: Maintains the last log entries in memory with configurable limits on entries and memory usage, ensuring efficient log management.
//   - Structured Logging: Utilizes the zap logging library for structured, high-performance logging.
//
// # Getting Started With Logging
//
//   1. Initialize the MarketDataClient with your API token.
//   2. Enable Debug mode for verbose logging during development.
//   3. Perform HTTP requests and utilize the logging features to monitor request details, responses, and rate limit usage.
//   4. Review the in-memory logs or structured log files to analyze HTTP interactions and troubleshoot issues.
//
// [comprehensive logging framework]: https://www.marketdata.app/docs/sdk/go/logging
package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	blue   = color.New(color.FgBlue).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	purple = color.New(color.FgMagenta).SprintFunc()
)

var (
	// logs holds a collection of HTTP request logs.
	logs *MarketDataLogs
)

// MarketDataLogs represents a collection of HTTP request logs.
// It provides methods to manipulate and retrieve log entries.
//
// # Methods
//
//   - String() string: Returns a string representation of all HTTP request logs.
//   - PrintLatest(): Prints the latest HTTP request log entry.
//   - LatestString() string : Gets the HTTP body response of the last log entry in string format.
type MarketDataLogs struct {
	// MaxLogEntries defines the maximum number of log entries that will be stored in memory.
	// Beyond this limit, older entries may be evicted or ignored.
	MaxEntries int

	// MemoryLimit specifies the maximum amount of memory (in bytes) that can be used for storing log entries.
	// This helps in preventing excessive memory usage by in-memory log storage.
	MemoryLimit int64

	// SuccessLogger is a zap logger used for logging successful operations.
	SuccessLogger *zap.Logger

	// ClientErrorLogger is a zap logger used for logging client-side errors.
	ClientErrorLogger *zap.Logger

	// ServerErrorLogger is a zap logger used for logging server-side errors.
	ServerErrorLogger *zap.Logger

	// Logs is the slice of LogEntry that contain all the request logs.
	Logs []LogEntry
}

// totalMemoryUsage calculates the total memory usage of all log entries.
//
// # Returns
//
//   - An int64 representing the total memory usage of all log entries in bytes.
func (h *MarketDataLogs) totalMemoryUsage() int64 {
	total := int64(0)
	for _, log := range h.Logs {
		total += log.memory
	}
	return total
}

// String returns a string representation of all HTTP request logs.
//
// # Returns
//
//   - A string representing all log entries.
func (h *MarketDataLogs) String() string {
	var sb strings.Builder
	for _, log := range h.Logs {
		sb.WriteString(log.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// LatestString returns the response of the last log entry in the MarketDataLogs.
//
// This method checks if there are any logs present. If there are no logs, it returns a message indicating that no logs are available.
// If logs are present, it calculates the index of the last log entry, accesses it, and returns its response.
//
// # Returns
//
//   - A string representing the response of the last log entry. If no logs are available, returns "No logs available".
func (h *MarketDataLogs) LatestString() string {
	// Step 2: Check if there are no logs
	if len(h.Logs) == 0 {
		// Return an appropriate response for no logs
		return "No logs available"
	}

	// Step 3: Calculate the index of the last log entry and access it
	lastLogIndex := len(h.Logs) - 1
	lastLog := h.Logs[lastLogIndex]

	// Step 4: Return the Response of the last log entry
	return lastLog.Response
}

// PrintLatest prints the latest HTTP request log entry.
func (h *MarketDataLogs) PrintLatest() {
	if len(h.Logs) == 0 {
		fmt.Println("No logs available")
	} else {
		fmt.Println(blue("Latest Log Entry:"))
		h.Logs[len(h.Logs)-1].PrettyPrint()
	}
}

// LogEntry represents a single HTTP request log entry.
// It includes detailed information about the request and response, such as headers, status code, and response body.
//
// # Methods
//
//   - String() string: Returns a string representation of the HTTP request log entry.
//   - PrettyPrint(): Prints a formatted representation of the HTTP request log entry.
type LogEntry struct {
	Timestamp         time.Time
	RequestHeaders    http.Header // The Request Headers
	ResponseHeaders   http.Header // The Response Headers
	RayID             string      // The Ray ID from the HTTP response
	Request           string      // The URL of the HTTP Request
	Status            int         // The status code of the response
	RateLimitConsumed int         // The number of requests consumed from the rate limit
	Delay             int64       // The time (in miliseconds) between the request and the server's response
	Response          string      // The server response
	memory            int64       // The amount of memory (in bytes) used by the log entry
}

// WriteToLog writes the log entry to the appropriate logger based on the HTTP response status.
//
// # Parameters
//
//   - debug: A boolean indicating whether to log as a debug message.
func (h LogEntry) writeToLog(debug bool) {
	var logger *zap.Logger
	var logMessage string

	// Try to parse the JSON response into a map
	var jsonResponse map[string]interface{}
	err := json.Unmarshal([]byte(h.Response), &jsonResponse)

	// If the parsing fails, log the response as a string
	var responseBody interface{}
	if err != nil {
		responseBody = h.Response
	} else {
		responseBody = jsonResponse
	}

	if h.Status >= 200 && h.Status < 300 {
		if debug {
			logger = logs.SuccessLogger
			logMessage = "Successful Request"
		}
	} else if h.Status >= 400 && h.Status < 500 {
		logger = logs.ClientErrorLogger
		logMessage = "Client Error"
	} else if h.Status >= 500 {
		logger = logs.ServerErrorLogger
		logMessage = "Server Error"
	}

	if logger != nil {
		logger.Info(logMessage,
			zap.String("cf_ray", h.RayID),
			zap.String("request_url", h.Request),
			zap.Int("ratelimit_consumed", h.RateLimitConsumed),
			zap.Int("response_code", h.Status),
			zap.Int64("delay_ms", h.Delay),
			zap.Any("request_headers", h.RequestHeaders),
			zap.Any("response_headers", h.ResponseHeaders),
			zap.Any("response_body", responseBody), // Log the parsed JSON response or the original string

		)
	}
}

// String returns a string representation of the HTTP request log entry.
//
// # Returns
//
//   - A string representing the log entry.
func (h LogEntry) String() string {
	return fmt.Sprintf("Timestamp: %v, Status: %d, Request: %s, Request Headers: %s, RayID: %s, RateLimitConsumed: %d, Delay: %dms, Response Headers: %s, Response: %s",
		h.Timestamp.Format("2006-01-02 15:04:05"), h.Status, h.Request, h.RequestHeaders, h.RayID, h.RateLimitConsumed, h.Delay, h.ResponseHeaders, h.Response)
}

// PrettyPrint prints a formatted representation of the HTTP request log entry.
func (h LogEntry) PrettyPrint() {
	fmt.Println(blue("Timestamp:"), h.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println(blue("Request:"), h.Request)
	fmt.Println(blue("Request Headers:"))
	h.printHeaders(h.RequestHeaders)
	fmt.Println(blue("Status:"), h.Status)
	fmt.Println(blue("Ray ID:"), h.RayID)
	fmt.Println(blue("Rate Limit Consumed:"), h.RateLimitConsumed)
	fmt.Println(blue("Delay:"), fmt.Sprintf("%dms", h.Delay))
	fmt.Println(blue("Response Headers:"))
	h.printHeaders(h.ResponseHeaders)
	fmt.Println(blue("Response:"), h.Response)
}

// printHeaders prints the HTTP headers in a formatted manner. Headers starting with "X-Api-Ratelimit" are highlighted.
//
// # Parameters
//
//   - headers: The HTTP headers to be printed.
func (h LogEntry) printHeaders(headers http.Header) {
	keys := make([]string, 0, len(headers))
	for name := range headers {
		keys = append(keys, name)
	}
	sort.Strings(keys) // Sort the keys alphabetically

	for _, name := range keys {
		values := headers[name]
		if strings.HasPrefix(name, "X-Api-Ratelimit") {
			fmt.Println(purple(name + ": " + strings.Join(values, ", ")))
		} else {
			fmt.Println(yellow(name+": "), strings.Join(values, ", "))
		}
	}
}

// memoryUsage calculates the memory usage of the log entry.
//
// # Returns
//
//   - An integer representing the memory usage of the log entry in bytes.
func (h LogEntry) memoryUsage() int {
	// Size of time.Time (24 bytes)
	timestampSize := 24

	// Size of string: size of string header (16 bytes) + length of string
	rayIDSize := 16 + len(h.RayID)
	requestSize := 16 + len(h.Request)
	responseSize := 16 + len(h.Response)

	// Size of int (4 bytes)
	statusSize := 4
	rateLimitConsumedSize := 4

	// Size of int64 (8 bytes)
	delaySize := 8
	memorySize := 8

	// Size of http.Header
	reqHeadersSize := h.headerSize(h.RequestHeaders)
	resHeadersSize := h.headerSize(h.ResponseHeaders)

	totalSize := timestampSize + rayIDSize + requestSize + statusSize + rateLimitConsumedSize + delaySize + responseSize + memorySize + reqHeadersSize + resHeadersSize

	return totalSize
}

// headerSize calculates the memory usage of HTTP headers.
//
// # Parameters
//
//   - header: The HTTP headers for which the memory usage is calculated.
//
// # Returns
//
//   - An integer representing the memory usage of the headers in bytes.
func (h LogEntry) headerSize(header http.Header) int {
	size := 0
	for key, values := range header {
		// Size of string: size of string header (16 bytes) + length of string
		keySize := 16 + len(key)
		for _, value := range values {
			valueSize := 16 + len(value)
			size += keySize + valueSize
		}
	}
	return size
}

// NewLogEntry creates a new instance of LogEntry with the provided parameters.
// This function initializes the log entry with details of the HTTP request and response,
// including timestamps, request and response headers, and other relevant information.
//
// # Parameters
//
//   - time.Time: The time at which the HTTP request was made.
//   - string: The rayID is a unique identifier for the request, typically used for tracing requests.
//   - string: The raw HTTP request URL as a string.
//   - int: rateLimitConsumed represents the amount of rate limit quota consumed by this request.
//   - int64: The delay experienced during the processing of the request, in milliseconds.
//   - int: The HTTP status code returned in the response.
//   - string: The body of the HTTP response in string format.
//   - http.Header: The HTTP headers of the request as http.Header.
//   - http.Header: The HTTP headers of the response as http.Header.
//
// # Returns
//
//   - LogEntry: An instance of LogEntry populated with the provided parameters and calculated memory usage.
func NewLogEntry(timestamp time.Time, rayID string, request string, rateLimitConsumed int, delay int64, status int, body string, reqHeaders http.Header, resHeaders http.Header) LogEntry {
	log := LogEntry{
		Timestamp:         timestamp,
		Status:            status,
		RayID:             rayID,
		RateLimitConsumed: rateLimitConsumed,
		Delay:             delay,
		Request:           request,
		Response:          body,
		RequestHeaders:    reqHeaders,
		ResponseHeaders:   resHeaders,
	}

	log.memory = int64(log.memoryUsage())

	return log
}

// addToLog adds a new HTTP request log entry to the MarketDataLogs.
//
// This method creates a new LogEntry entry based on the provided parameters and appends it to the MarketDataLogs.
// After adding a new log entry, it trims the log to ensure the total memory usage and the number of log entries are below their limits.
//
// # Parameters
//
//   - h *MarketDataLogs: A pointer to the MarketDataLogs to which the new log entry will be added.
//   - timestamp time.Time: The timestamp of the HTTP request.
//   - rayID string: The unique identifier for the request.
//   - request string: The URL of the HTTP request.
//   - rateLimitConsumed int: The amount of rate limit consumed by the request.
//   - delay int64: The delay experienced during the request, in milliseconds.
//   - status int: The HTTP status code of the response.
//   - body string: The body of the HTTP response.
//   - reqHeaders http.Header: The HTTP headers of the request.
//   - resHeaders http.Header: The HTTP headers of the response.
//
// # Returns
//
//   - *LogEntry: A pointer to the newly added LogEntry entry. Returns nil if the log entry is not added.
func addToLog(h *MarketDataLogs, timestamp time.Time, rayID string, request string, rateLimitConsumed int, delay int64, status int, body string, reqHeaders http.Header, resHeaders http.Header) *LogEntry {
	if request == "https://api.marketdata.app/user/" {
		// If the URL starts with https://api.marketdata.app/user/ do not add it to the log.
		// Just return without doing anything in this case.
		return nil
	}

	log := NewLogEntry(timestamp, rayID, request, rateLimitConsumed, delay, status, body, reqHeaders, resHeaders)

	h.Logs = append(h.Logs, log)

	// Trim the log to ensure the total memory usage and the number of log entries are below their limits
	h.trimLog()

	// Return a pointer to the new log entry
	return &h.Logs[len(h.Logs)-1]
}

// trimLog trims the MarketDataLogs to ensure that the total memory usage and the number of log entries do not exceed their respective limits.
// It iteratively removes the oldest log entry until the memory usage is below the MemoryLimit and the number of entries is less than or equal to MaxLogEntries.
func (h *MarketDataLogs) trimLog() {
	// While the total memory usage is above the limit or there are too many log entries, remove the oldest log entry
	for (h.totalMemoryUsage() > logs.MemoryLimit || len(h.Logs) > logs.MaxEntries) && len(h.Logs) > 0 {
		h.Logs = h.Logs[1:]
	}
}

// init initializes the logging system for the application.
//
// This function performs the following operations:
//   - Initializes the Logs variable with an empty MarketDataLogs.
//   - Checks if the logs directory exists, and creates it if it does not.
//   - Opens or creates the success, client error, and server error log files.
//   - Sets up a zapcore.Core for each log file to enable structured logging.
//
// The log files are named success.log, client_error.log, and server_error.log respectively.
// Each log file is opened with append mode, so new log entries are added to the end of the file.
// The logging level for all log files is set to InfoLevel, and the logs are encoded in JSON format.
// The time encoding for log entries is set to ISO8601 format.
func init() {
	// Initialize the Logs variable
	logs = &MarketDataLogs{
		Logs:        []LogEntry{},
		MemoryLimit: 10 * 1048576,
		MaxEntries:  100000,
	}

	// Create the logs directory if it does not exist
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err = os.MkdirAll("logs", 0755)
		if err != nil {
			log.Fatalf("Failed to create logs directory: %v", err)
		}
	}

	// Open the log files. If they do not exist, create them.
	successLogFile, err := os.OpenFile("logs/success.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open success log file: %v", err)
	}
	defer successLogFile.Close() // Ensure the file is closed when no longer needed

	clientErrorLogFile, err := os.OpenFile("logs/client_error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open client error log file: %v", err)
	}
	defer clientErrorLogFile.Close() // Ensure the file is closed when no longer needed

	serverErrorLogFile, err := os.OpenFile("logs/server_error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open server error log file: %v", err)
	}
	defer serverErrorLogFile.Close() // Ensure the file is closed when no longer needed

	// Create a zapcore.Core that writes to the log files
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	successCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(successLogFile),
		zap.InfoLevel,
	)

	clientErrorCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(clientErrorLogFile),
		zap.InfoLevel,
	)

	serverErrorCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(serverErrorLogFile),
		zap.InfoLevel,
	)

	// Create a zap.Logger from the Core
	logs.SuccessLogger = zap.New(successCore)
	logs.ClientErrorLogger = zap.New(clientErrorCore)
	logs.ServerErrorLogger = zap.New(serverErrorCore)
}

// GetLogs retrieves a pointer to the MarketDataLogs instance, allowing access to the logs collected during HTTP requests.
// This method is primarily used for debugging and monitoring purposes, providing insights into the HTTP request lifecycle and any issues that may have occurred.
//
// # Returns
//
//   - *MarketDataLogs: A pointer to the MarketDataLogs instance containing logs of HTTP requests.
func GetLogs() *MarketDataLogs {
	return logs
}
