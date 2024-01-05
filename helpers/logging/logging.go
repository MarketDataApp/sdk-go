package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	//debugModeLogger = log.New(os.Stdout, "", 0) // 0 turns off all flags, including the default timestamp flag
	blue            = color.New(color.FgBlue).SprintFunc()
	yellow          = color.New(color.FgYellow).SprintFunc()
	//purple          = color.New(color.FgMagenta).SprintFunc()
)

var (
	SuccessLogger     *zap.Logger
	ClientErrorLogger *zap.Logger
	ServerErrorLogger *zap.Logger
	Logs              *HttpRequestLogs
	MaxLogEntries     = 100000 // MaxLogEntries is the maximum number of log entries that will be stored in memory.

)

type HttpRequestLogs struct {
	Logs []HttpRequestLog
}

type HttpRequestLog struct {
	Timestamp         time.Time
	RayID             string // The Ray ID from the HTTP response
	URL               string // The URL of the HTTP Request
	Status            int    // The status code of the response
	RateLimitConsumed int    // The number of requests consumed from the rate limit
	Delay             int64  // The time (in miliseconds) between the request and the server's response
}

func (h HttpRequestLog) String() string {
	return fmt.Sprintf("Timestamp: %v, RayID: %s, RateLimitConsumed: %d, Delay: %dms, URL: %s",
		h.Timestamp.Format("2006-01-02 15:04:05"), h.RayID, h.RateLimitConsumed, h.Delay, h.URL)
}

func (h *HttpRequestLogs) String() string {
	var sb strings.Builder
	for _, log := range h.Logs {
		sb.WriteString(log.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

func (h *HttpRequestLogs) PrintLatest() {
	if len(h.Logs) == 0 {
		fmt.Println("No logs available")
	} else {
		fmt.Println(blue("Latest Log Entry:"))
		fmt.Println(yellow("Timestamp:"), h.Logs[len(h.Logs)-1].Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Println(yellow("RayID:"), h.Logs[len(h.Logs)-1].RayID)
		fmt.Println(yellow("RateLimitConsumed:"), h.Logs[len(h.Logs)-1].RateLimitConsumed)
		fmt.Println(yellow("Delay:"), fmt.Sprintf("%dms", h.Logs[len(h.Logs)-1].Delay))
		fmt.Println(yellow("RequestURL:"), h.Logs[len(h.Logs)-1].URL)
	}
}

func (h *HttpRequestLogs) AddToLog(timestamp time.Time, rayID string, url string, rateLimitConsumed int, delay int64) {
	log := HttpRequestLog{
		Timestamp:         timestamp,
		RayID:             rayID,
		URL:               url,
		RateLimitConsumed: rateLimitConsumed,
		Delay:             delay,
	}

	if MaxLogEntries != 0 && len(h.Logs) >= MaxLogEntries {
		// Remove the oldest entry
		h.Logs = h.Logs[1:]
	}

	h.Logs = append(h.Logs, log)
}

func init() {
	// Initialize the Logs variable
	Logs = &HttpRequestLogs{
		Logs: []HttpRequestLog{},
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

	clientErrorLogFile, err := os.OpenFile("logs/client_error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open client error log file: %v", err)
	}

	serverErrorLogFile, err := os.OpenFile("logs/server_error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open server error log file: %v", err)
	}

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
	SuccessLogger = zap.New(successCore)
	ClientErrorLogger = zap.New(clientErrorCore)
	ServerErrorLogger = zap.New(serverErrorCore)
}
