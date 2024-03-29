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
//  1. Initialize the MarketDataClient with your API token.
//  2. Enable Debug mode for verbose logging during development.
//  3. Perform HTTP requests and utilize the logging features to monitor request details, responses, and rate limit usage.
//  4. Review the in-memory logs or structured log files to analyze HTTP interactions and troubleshoot issues.
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

	"github.com/MarketDataApp/sdk-go/helpers/dates"
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

	// MemoryLimit specifies the maximum amount of memory (in bytes) that can be used for
	// storing log entries.  This helps in preventing excessive memory usage by in-memory
	// log storage.
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
func (l *MarketDataLogs) totalMemoryUsage() int64 {
	total := int64(0)
	for _, log := range l.Logs {
		total += log.memory
	}
	return total
}

// String returns a string representation of all HTTP request logs.
//
// # Returns
//
//   - string: A string representing all log entries, with log entry numbers starting at 0.
//
// # Example
//
//    MarketDataLogs{
//      LogEntry[0]: {Timestamp: 2024-02-19 15:16:30 -05:00, Status: 200, Request: https://api.marketdata.app/v1/stocks/candles/4H/AAPL/?from=2023-01-01&to=2023-01-04, RequestHeaders: map[Authorization:[Bearer **********************************************************8IH6] User-Agent:[sdk-go/1.1.0]], RayID: 8581301bc9e974ca-MIA, RateLimitConsumed: 0, Delay: 1002ms, Response Headers: map[Allow:[GET, HEAD, OPTIONS] Alt-Svc:[h3=":443"; ma=86400] Cf-Cache-Status:[DYNAMIC] Cf-Ray:[8581301bc9e974ca-MIA] Content-Type:[application/json] Cross-Origin-Opener-Policy:[same-origin] Date:[Mon, 19 Feb 2024 20:16:30 GMT] Nel:[{"success_fraction":0,"report_to":"cf-nel","max_age":604800}] Referrer-Policy:[same-origin] Report-To:[{"endpoints":[{"url":"https:\/\/a.nel.cloudflare.com\/report\/v3?s=ar7qRMEAxiHJcyRqlRlZvfzqv80inCp91LwdsxTfo51%2FSmdqz6NoadysPyFlMZKtDMG1IsDllX0GXXPjNsvjg4gQBr%2FLJ8G5Ad1z5lDXfhqrSee7umXzVf7dYmAFwq4x4sPMANE%3D"}],"group":"cf-nel","max_age":604800}] Server:[cloudflare] Vary:[Accept, Origin] X-Api-Ratelimit-Consumed:[0] X-Api-Ratelimit-Limit:[100000] X-Api-Ratelimit-Remaining:[99973] X-Api-Ratelimit-Reset:[1708439400] X-Api-Response-Log-Id:[88078705] X-Content-Type-Options:[nosniff] X-Frame-Options:[DENY]], Response: {"s":"ok","t":[1672756200,1672770600,1672842600,1672857000],"o":[130.28,124.6699,126.89,127.265],"h":[130.9,125.42,128.6557,127.87],"l":[124.19,124.17,125.08,125.28],"c":[124.6499,125.05,127.2601,126.38],"v":[64411753,30727802,49976607,28870878]}}
//    }
func (l *MarketDataLogs) String() string {
	var sb strings.Builder
	sb.WriteString("MarketDataLogs{\n")
	for i, log := range l.Logs {
		sb.WriteString(fmt.Sprintf("  LogEntry[%d]: %s\n", i, log.detailedString(false)))
	}
	sb.WriteString("}")
	return sb.String()
}
// LatestString returns the response of the last log entry in the MarketDataLogs.
//
// This method checks if there are any logs present. If there are no logs, it returns a message indicating that no logs are available.
// If logs are present, it calculates the index of the last log entry, accesses it, and returns its response.
//
// # Returns
//
//   - string: A string representing the response of the last log entry. If no logs are available, returns "No logs available".
func (l *MarketDataLogs) LatestString() string {
	// Step 2: Check if there are no logs
	if len(l.Logs) == 0 {
		// Return an appropriate response for no logs
		return "No logs available"
	}

	// Step 3: Calculate the index of the last log entry and access it
	lastLogIndex := len(l.Logs) - 1
	lastLog := l.Logs[lastLogIndex]

	// Step 4: Return the Response of the last log entry
	return lastLog.Response
}

// PrintLatest prints the latest HTTP request log entry.
func (l *MarketDataLogs) PrintLatest() {
	if len(l.Logs) == 0 {
		fmt.Println("No logs available")
	} else {
		fmt.Println(blue("Latest Log Entry:"))
		l.Logs[len(l.Logs)-1].PrettyPrint()
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
	Delay             int64       // The time (in milliseconds) between the request and the server's response
	Response          string      // The server response
	memory            int64       // The amount of memory (in bytes) used by the log entry
}

// writeToLog writes the log entry to the appropriate logger based on the HTTP response status.
//
// # Parameters
//
//   - debug: A boolean indicating whether to log as a debug message.
func (l LogEntry) writeToLog(debug bool) {
	var logger *zap.Logger
	var logMessage string

	// Try to parse the JSON response into a map
	var jsonResponse map[string]interface{}
	err := json.Unmarshal([]byte(l.Response), &jsonResponse)

	// If the parsing fails, log the response as a string
	var responseBody interface{}
	if err != nil {
		responseBody = l.Response
	} else {
		responseBody = jsonResponse
	}

	if l.Status >= 200 && l.Status < 300 {
		if debug {
			logger = logs.SuccessLogger
			logMessage = "Successful Request"
		}
	} else if l.Status >= 400 && l.Status < 500 {
		logger = logs.ClientErrorLogger
		logMessage = "Client Error"
	} else if l.Status >= 500 {
		logger = logs.ServerErrorLogger
		logMessage = "Server Error"
	}

	if logger != nil {
		logger.Info(logMessage,
			zap.String("cf_ray", l.RayID),
			zap.String("request_url", l.Request),
			zap.Int("ratelimit_consumed", l.RateLimitConsumed),
			zap.Int("response_code", l.Status),
			zap.Int64("delay_ms", l.Delay),
			zap.Any("request_headers", l.RequestHeaders),
			zap.Any("response_headers", l.ResponseHeaders),
			zap.Any("response_body", responseBody), // Log the parsed JSON response or the original string

		)
	}
}

// String returns a string representation of the HTTP request log entry.
//
// # Returns
//
//   - string: A string representing the log entry.
//
// # Example
//
//    LogEntry{Timestamp: 2024-02-19 14:53:34 -05:00, Status: 200, Request: https://api.marketdata.app/v1/stocks/candles/4H/AAPL/?from=2023-01-01&to=2023-01-04, RequestHeaders: map[Authorization:[Bearer **********************************************************L06F] User-Agent:[sdk-go/1.1.0]], RayID: 85810e82de8e7438-MIA, RateLimitConsumed: 0, Delay: 893ms, Response Headers: map[Allow:[GET, HEAD, OPTIONS] Alt-Svc:[h3=":443"; ma=86400] Cf-Cache-Status:[DYNAMIC] Cf-Ray:[85810e82de8e7438-MIA] Content-Type:[application/json] Cross-Origin-Opener-Policy:[same-origin] Date:[Mon, 19 Feb 2024 19:53:34 GMT] Nel:[{"success_fraction":0,"report_to":"cf-nel","max_age":604800}] Referrer-Policy:[same-origin] Report-To:[{"endpoints":[{"url":"https:\/\/a.nel.cloudflare.com\/report\/v3?s=r18O66yjJ%2FxXqOnCb3a76wBgpZaCbhJcot%2Bfgl1oHna2LigHHAYRaXg8dNLiJYHes0ezAaIdXLhVNGQBo%2FBAte6%2ByNcaZku5cV19FPyiD2%2BKXJeEtnFN6pJUsUA77sxk%2FfWxOFU%3D"}],"group":"cf-nel","max_age":604800}] Server:[cloudflare] Vary:[Accept, Origin] X-Api-Ratelimit-Consumed:[0] X-Api-Ratelimit-Limit:[100000] X-Api-Ratelimit-Remaining:[99973] X-Api-Ratelimit-Reset:[1708439400] X-Api-Response-Log-Id:[88061028] X-Content-Type-Options:[nosniff] X-Frame-Options:[DENY]], Response: {"s":"ok","t":[1672756200,1672770600,1672842600,1672857000],"o":[130.28,124.6699,126.89,127.265],"h":[130.9,125.42,128.6557,127.87],"l":[124.19,124.17,125.08,125.28],"c":[124.6499,125.05,127.2601,126.38],"v":[64411753,30727802,49976607,28870878]}}
//
func (l LogEntry) String() string {
	return l.detailedString(true)
}

// PrettyPrint prints a formatted representation of the HTTP request log entry.
func (l LogEntry) PrettyPrint() {
	fmt.Println(blue("Timestamp:"), l.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println(blue("Request:"), l.Request)
	fmt.Println(blue("Request Headers:"))
	l.printHeaders(l.RequestHeaders)
	fmt.Println(blue("Status:"), l.Status)
	fmt.Println(blue("Ray ID:"), l.RayID)
	fmt.Println(blue("Rate Limit Consumed:"), l.RateLimitConsumed)
	fmt.Println(blue("Delay:"), fmt.Sprintf("%dms", l.Delay))
	fmt.Println(blue("Response Headers:"))
	l.printHeaders(l.ResponseHeaders)
	fmt.Println(blue("Response:"), l.Response)
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
func (l LogEntry) memoryUsage() int {
	// Size of time.Time (24 bytes)
	timestampSize := 24

	// Size of string: size of string header (16 bytes) + length of string
	rayIDSize := 16 + len(l.RayID)
	requestSize := 16 + len(l.Request)
	responseSize := 16 + len(l.Response)

	// Size of int (4 bytes)
	statusSize := 4
	rateLimitConsumedSize := 4

	// Size of int64 (8 bytes)
	delaySize := 8
	memorySize := 8

	// Size of http.Header
	reqHeadersSize := l.headerSize(l.RequestHeaders)
	resHeadersSize := l.headerSize(l.ResponseHeaders)

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

// newLogEntry creates a new instance of LogEntry with the provided parameters.
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
func newLogEntry(timestamp time.Time, rayID string, request string, rateLimitConsumed int, delay int64, status int, body string, reqHeaders http.Header, resHeaders http.Header) LogEntry {
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
//   - l *MarketDataLogs: A pointer to the MarketDataLogs to which the new log entry will be added.
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
func addToLog(l *MarketDataLogs, timestamp time.Time, rayID string, request string, rateLimitConsumed int, delay int64, status int, body string, reqHeaders http.Header, resHeaders http.Header) *LogEntry {
	if request == "https://api.marketdata.app/user/" {
		// If the URL starts with https://api.marketdata.app/user/ do not add it to the log.
		// Just return without doing anything in this case.
		return nil
	}

	log := newLogEntry(timestamp, rayID, request, rateLimitConsumed, delay, status, body, reqHeaders, resHeaders)

	l.Logs = append(l.Logs, log)

	// Trim the log to ensure the total memory usage and the number of log entries are below their limits
	l.trimLog()

	// Return a pointer to the new log entry
	return &l.Logs[len(l.Logs)-1]
}

// trimLog trims the MarketDataLogs to ensure that the total memory usage and the number of log entries do not exceed their respective limits.
// It iteratively removes the oldest log entry until the memory usage is below the MemoryLimit and the number of entries is less than or equal to MaxLogEntries.
func (l *MarketDataLogs) trimLog() {
	// While the total memory usage is above the limit or there are too many log entries, remove the oldest log entry
	for (l.totalMemoryUsage() > logs.MemoryLimit || len(l.Logs) > logs.MaxEntries) && len(l.Logs) > 0 {
		l.Logs = l.Logs[1:]
	}
}

// detailedString returns a string representation of the HTTP request log entry, with an option to include the struct name.
//
// # Parameters
//
//   - includeStructName: A boolean indicating whether to prefix the output with "LogEntry".
//
// # Returns
//
//   - string: A string representing the log entry, optionally prefixed with "LogEntry".
func (l LogEntry) detailedString(includeStructName bool) string {
	prefix := ""
	if includeStructName {
		prefix = "LogEntry"
	}
	return fmt.Sprintf("%s{Timestamp: %v, Status: %d, Request: %s, RequestHeaders: %s, RayID: %s, RateLimitConsumed: %d, Delay: %dms, Response Headers: %s, Response: %s}",
		prefix, dates.TimeString(l.Timestamp), l.Status, l.Request, l.RequestHeaders, l.RayID, l.RateLimitConsumed, l.Delay, l.ResponseHeaders, l.Response)
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
