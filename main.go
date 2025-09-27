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
	TypeStart  = "0"
	TypeEnd    = "1"
	TypePause  = "2"
	TypeResume = "3"
	LogFile    = "pomo_log.csv"
)

type Entry struct {
	Timestamp time.Time
	TaskName  string
	Type      string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: pomo <command> [args]")
		fmt.Println("Commands: start <task name>, stop, pause, resume, now, stats")
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
	case "pause":
		pauseTask()
	case "resume":
		resumeTask()
	case "now":
		showCurrentTask()
	case "stats":
		showStats()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Commands: start <task name>, stop, pause, resume, now, stats")
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

	if lastEntry != nil && (lastEntry.Type == TypeStart || lastEntry.Type == TypeResume) {
		fmt.Printf("Task '%s' is already running. Stop it first.\n", lastEntry.TaskName)
		os.Exit(1)
	}

	if lastEntry != nil && lastEntry.Type == TypePause {
		fmt.Printf("Task '%s' is paused. Resume it first or stop it to start a new task.\n", lastEntry.TaskName)
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

func pauseTask() {
	lastEntry, err := getLastEntry()
	if err != nil {
		fmt.Printf("Error reading log: %v\n", err)
		os.Exit(1)
	}

	if lastEntry == nil {
		fmt.Println("No task is currently running")
		os.Exit(1)
	}

	if lastEntry.Type == TypeEnd {
		fmt.Println("No task is currently running")
		os.Exit(1)
	}

	if lastEntry.Type == TypePause {
		fmt.Printf("Task '%s' is already paused\n", lastEntry.TaskName)
		os.Exit(1)
	}

	entry := Entry{
		Timestamp: time.Now(),
		TaskName:  lastEntry.TaskName,
		Type:      TypePause,
	}

	if err := writeEntry(entry); err != nil {
		fmt.Printf("Error writing to log: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Paused task '%s'\n", lastEntry.TaskName)
}

func resumeTask() {
	lastEntry, err := getLastEntry()
	if err != nil {
		fmt.Printf("Error reading log: %v\n", err)
		os.Exit(1)
	}

	if lastEntry == nil {
		fmt.Println("No task is paused")
		os.Exit(1)
	}

	if lastEntry.Type != TypePause {
		fmt.Println("No task is paused")
		os.Exit(1)
	}

	entry := Entry{
		Timestamp: time.Now(),
		TaskName:  lastEntry.TaskName,
		Type:      TypeResume,
	}

	if err := writeEntry(entry); err != nil {
		fmt.Printf("Error writing to log: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Resumed task '%s'\n", lastEntry.TaskName)
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

	if lastEntry.Type == TypePause {
		fmt.Printf("Current task: '%s' (paused)\n", lastEntry.TaskName)
		return
	}

	duration := time.Since(lastEntry.Timestamp)
	status := "running"
	if lastEntry.Type == TypeResume {
		status = "resumed"
	}
	fmt.Printf("Current task: '%s' (%s for %s)\n", lastEntry.TaskName, status, formatDuration(duration))
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
		if entries[i].Type == TypeStart || entries[i].Type == TypeResume {
			// Find corresponding pause or end
			var segmentDuration time.Duration
			found := false

			for j := i + 1; j < len(entries); j++ {
				if entries[j].TaskName == entries[i].TaskName {
					if entries[j].Type == TypePause || entries[j].Type == TypeEnd {
						segmentDuration = entries[j].Timestamp.Sub(entries[i].Timestamp)
						found = true

						// If this is an end, mark task as completed
						if entries[j].Type == TypeEnd {
							completedTasks++
							date := entries[i].Timestamp.Format("2006-01-02")
							dailyTasks[date]++

							if date == today {
								todayTasks++
							}
						}
						break
					}
				}
			}

			if found {
				taskDurations[entries[i].TaskName] += segmentDuration
				totalDuration += segmentDuration

				date := entries[i].Timestamp.Format("2006-01-02")
				if date == today {
					todayDuration += segmentDuration
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
