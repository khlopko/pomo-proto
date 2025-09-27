package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	TypeStart = "0"
	TypeEnd   = "1"
	LogFile   = "pomo_log.csv"
)

type Entry struct {
	Timestamp time.Time
	TaskName  string
	Type      string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: pomo <command> [args]")
		fmt.Println("Commands: start <task name>, stop, now, stats")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "start":
		if len(os.Args) < 3 {
			fmt.Println("Usage: pomo start <task name>")
			os.Exit(1)
		}
		taskName := strings.Join(os.Args[2:], " ")
		startTask(taskName)
	case "stop":
		stopTask()
	case "now":
		showCurrentTask()
	case "stats":
		showStats()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Commands: start <task name>, stop, now, stats")
		os.Exit(1)
	}
}

func getLogFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "."+LogFile)
}

func writeEntry(entry Entry) error {
	logPath := getLogFilePath()

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	record := []string{
		entry.Timestamp.Format(time.RFC3339),
		entry.TaskName,
		entry.Type,
	}

	return writer.Write(record)
}

func readEntries() ([]Entry, error) {
	logPath := getLogFilePath()

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, record := range records {
		if len(record) != 3 {
			continue
		}

		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			continue
		}

		entries = append(entries, Entry{
			Timestamp: timestamp,
			TaskName:  record[1],
			Type:      record[2],
		})
	}

	return entries, nil
}

func getLastEntry() (*Entry, error) {
	entries, err := readEntries()
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	return &entries[len(entries)-1], nil
}

func startTask(taskName string) {
	lastEntry, err := getLastEntry()
	if err != nil {
		fmt.Printf("Error reading log: %v\n", err)
		os.Exit(1)
	}

	if lastEntry != nil && lastEntry.Type == TypeStart {
		fmt.Printf("Task '%s' is already running. Stop it first.\n", lastEntry.TaskName)
		os.Exit(1)
	}

	entry := Entry{
		Timestamp: time.Now(),
		TaskName:  taskName,
		Type:      TypeStart,
	}

	if err := writeEntry(entry); err != nil {
		fmt.Printf("Error writing to log: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Started tracking '%s'\n", taskName)
}

func stopTask() {
	lastEntry, err := getLastEntry()
	if err != nil {
		fmt.Printf("Error reading log: %v\n", err)
		os.Exit(1)
	}

	if lastEntry == nil || lastEntry.Type == TypeEnd {
		fmt.Println("No task is currently running")
		os.Exit(1)
	}

	entry := Entry{
		Timestamp: time.Now(),
		TaskName:  lastEntry.TaskName,
		Type:      TypeEnd,
	}

	if err := writeEntry(entry); err != nil {
		fmt.Printf("Error writing to log: %v\n", err)
		os.Exit(1)
	}

	duration := entry.Timestamp.Sub(lastEntry.Timestamp)
	fmt.Printf("Stopped tracking '%s' (duration: %s)\n", lastEntry.TaskName, formatDuration(duration))
}

func showCurrentTask() {
	lastEntry, err := getLastEntry()
	if err != nil {
		fmt.Printf("Error reading log: %v\n", err)
		os.Exit(1)
	}

	if lastEntry == nil || lastEntry.Type == TypeEnd {
		fmt.Println("No task is currently running")
		return
	}

	duration := time.Since(lastEntry.Timestamp)
	fmt.Printf("Current task: '%s' (running for %s)\n", lastEntry.TaskName, formatDuration(duration))
}

func showStats() {
	entries, err := readEntries()
	if err != nil {
		fmt.Printf("Error reading log: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No tasks recorded yet")
		return
	}

	// Calculate statistics
	taskDurations := make(map[string]time.Duration)
	dailyTasks := make(map[string]int)
	var totalDuration time.Duration
	var completedTasks int

	today := time.Now().Format("2006-01-02")
	var todayDuration time.Duration
	var todayTasks int

	for i := 0; i < len(entries); i++ {
		if entries[i].Type == TypeStart {
			// Find corresponding end
			var endTime time.Time
			found := false
			for j := i + 1; j < len(entries); j++ {
				if entries[j].Type == TypeEnd && entries[j].TaskName == entries[i].TaskName {
					endTime = entries[j].Timestamp
					found = true
					break
				}
			}

			if found {
				duration := endTime.Sub(entries[i].Timestamp)
				taskDurations[entries[i].TaskName] += duration
				totalDuration += duration
				completedTasks++

				date := entries[i].Timestamp.Format("2006-01-02")
				dailyTasks[date]++

				if date == today {
					todayDuration += duration
					todayTasks++
				}
			}
		}
	}

	// Calculate averages
	var avgDurationOverall time.Duration
	if completedTasks > 0 {
		avgDurationOverall = totalDuration / time.Duration(completedTasks)
	}

	var avgDurationToday time.Duration
	if todayTasks > 0 {
		avgDurationToday = todayDuration / time.Duration(todayTasks)
	}

	// Calculate average tasks per day
	var avgTasksPerDay float64
	if len(dailyTasks) > 0 {
		totalTasksAllDays := 0
		for _, count := range dailyTasks {
			totalTasksAllDays += count
		}
		avgTasksPerDay = float64(totalTasksAllDays) / float64(len(dailyTasks))
	}

	// Output statistics
	fmt.Println("=== Pomodoro Statistics ===")
	fmt.Printf("Total completed tasks: %d\n", completedTasks)
	fmt.Printf("Average tasks per day: %.1f\n", avgTasksPerDay)
	fmt.Printf("Average task duration today: %s\n", formatDuration(avgDurationToday))
	fmt.Printf("Average task duration overall: %s\n", formatDuration(avgDurationOverall))
	fmt.Printf("Tasks completed today: %d\n", todayTasks)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
