package linuxcollector

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aaronlmathis/gosight/agent/internal/config"
	"github.com/aaronlmathis/gosight/shared/model"
	"github.com/aaronlmathis/gosight/shared/utils"
	"github.com/coreos/go-systemd/v22/sdjournal"
)

// JournaldTailCollector streams only new log entries since start.
type JournaldCollector struct {
	journal *sdjournal.Journal
	Config  *config.Config
}

// Name returns the name of the collector.
func (j *JournaldCollector) Name() string {
	return "journald"
}

// NewJournaldCollector initializes a new JournaldCollector.
func NewJournaldCollector(cfg *config.Config) *JournaldCollector {
	j, err := sdjournal.NewJournal()
	if err != nil {
		// Optional: log or panic if initialization fails
		utils.Debug("Failed to open systemd journal: %v", err)
	}

	for _, prio := range []string{"0", "1", "2", "3", "4"} {
		_ = j.AddMatch("PRIORITY=" + prio)
		_ = j.AddDisjunction()
	}

	// Seek to end to skip historical logs
	if err := j.SeekTail(); err != nil {
		utils.Debug("failed to seek to tail: %v\n", err)

	}
	// Skip current tail entry (so we get only new logs)
	_, err = j.Next()
	if err != nil {
		utils.Debug("failed to get next entry: %v\n", err)

	}
	return &JournaldCollector{
		Config:  cfg,
		journal: j,
	}
}

// Collect retrieves log entries from the journal.
// It returns a slice of log entries, each containing a timestamp, level, message, and other metadata.
// The method blocks until it has collected a batch of logs or the context is canceled.
// It also handles the batching of logs based on the configured batch size and interval.
func (j *JournaldCollector) Collect(ctx context.Context) ([][]model.LogEntry, error) {
	var allBatches [][]model.LogEntry
	var current []model.LogEntry

	batchSize := j.Config.Agent.LogCollection.BatchSize
	batchInterval := time.Duration(j.Config.Agent.Interval) * time.Millisecond
	maxSize := j.Config.Agent.LogCollection.MessageMax
	start := time.Now()

	// Wait for new logs up to 2 seconds
	j.journal.Wait(2 * time.Second)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			n, err := j.journal.Next()
			if err != nil || n == 0 {
				break loop
			}
			entry, err := j.journal.GetEntry()
			if err != nil {
				continue
			}
			if entry.Fields["SYSLOG_IDENTIFIER"] == "kernel" {
				continue
			}

			log := buildLogEntry(entry, maxSize)
			current = append(current, log)

			if len(current) >= batchSize || time.Since(start) >= batchInterval {
				allBatches = append(allBatches, current)
				current = nil
				start = time.Now()
			}
		}
	}

	if len(current) > 0 {
		allBatches = append(allBatches, current)
	}

	if len(allBatches) == 0 {
		return nil, nil
	}
	return allBatches, nil
}

// mapPriorityToLevel maps systemd journal priority levels to log levels.
// It converts the priority string to a corresponding log level string.

func mapPriorityToLevel(priority string) string {
	switch priority {
	case "0", "1", "2":
		return "error"
	case "3":
		return "warn"
	case "4":
		return "info"
	case "5":
		return "notice"
	case "6":
		return "debug"
	default:
		return "unknown"
	}
}

// buildLogEntry constructs a LogEntry from a systemd journal entry.
// It extracts the timestamp, level, message, source, and other metadata from the entry.
// It also truncates the message if it exceeds the specified maximum size and sanitizes it to ensure valid UTF-8 encoding.
// The function returns a LogEntry struct containing all the relevant information.
// It also filters out unwanted fields and includes only the specified fields in the LogEntry.
func buildLogEntry(entry *sdjournal.JournalEntry, maxSize int) model.LogEntry {
	timestamp := time.Unix(0, int64(entry.RealtimeTimestamp)*int64(time.Microsecond))
	msg := entry.Fields["MESSAGE"]
	if len(msg) > maxSize {
		msg = msg[:maxSize] + " [truncated]"
	}

	if !utf8.ValidString(msg) {
		msg = sanitizeUTF8(msg) // or fallback
	}
	source := entry.Fields["SYSLOG_IDENTIFIER"]
	if source == "" {
		source = "unknown"
	}
	category := entry.Fields["_SYSTEMD_UNIT"]
	if category == "" {
		category = "unknown"
	}
	// Filtered fields
	wanted := []string{"_SYSTEMD_UNIT", "_EXE", "_CMDLINE", "_PID", "_UID", "MESSAGE_ID", "SYSLOG_IDENTIFIER", "CONTAINER_ID", "CONTAINER_NAME"}
	fields := make(map[string]string)
	for _, k := range wanted {
		if v, ok := entry.Fields[k]; ok {
			fields[k] = v
		}
	}

	return model.LogEntry{
		Timestamp: timestamp,
		Level:     mapPriorityToLevel(entry.Fields["PRIORITY"]),
		Message:   msg,
		Source:    source,
		Category:  category,
		PID:       parsePID(entry.Fields["_PID"]),
		Fields:    fields,
		Tags: map[string]string{
			"unit":           category,
			"container_id":   entry.Fields["CONTAINER_ID"],
			"container_name": entry.Fields["CONTAINER_NAME"],
		},
		Meta: &model.LogMeta{
			Platform:      "journald",
			AppName:       source,
			ContainerID:   entry.Fields["CONTAINER_ID"],
			ContainerName: entry.Fields["CONTAINER_NAME"],
			Unit:          category,
			Service:       entry.Fields["SYSLOG_IDENTIFIER"],
			EventID:       entry.Fields["MESSAGE_ID"],
			User:          entry.Fields["_UID"],
			Executable:    entry.Fields["_EXE"],
			Path:          entry.Fields["_CMDLINE"],
		},
	}
}

// parsePID converts a string representation of a PID to an integer.
func parsePID(pidStr string) int {
	pid, _ := strconv.Atoi(pidStr)
	return pid
}

// sanitizeUTF8 ensures that the string is valid UTF-8.
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "�")
}
