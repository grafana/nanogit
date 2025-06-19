package performance

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

// MetricsCollector handles performance metrics collection and reporting
type MetricsCollector struct {
	results []BenchmarkResult
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		results: make([]BenchmarkResult, 0),
	}
}

// RecordBenchmark records a benchmark result
func (m *MetricsCollector) RecordBenchmark(result BenchmarkResult) {
	m.results = append(m.results, result)
}

// RecordOperation is a helper to time and record an operation
func (m *MetricsCollector) RecordOperation(client, operation, scenario, repoSize string, fileCount int, fn func() error) {
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)
	memUsed := int64(memAfter.Alloc - memBefore.Alloc)

	result := BenchmarkResult{
		Client:     client,
		Operation:  operation,
		Scenario:   scenario,
		Duration:   duration,
		MemoryUsed: memUsed,
		Success:    err == nil,
		RepoSize:   repoSize,
		FileCount:  fileCount,
	}

	if err != nil {
		result.Error = err.Error()
	}

	m.RecordBenchmark(result)
}

// GetResults returns all recorded results
func (m *MetricsCollector) GetResults() []BenchmarkResult {
	return m.results
}

// GenerateReport generates a performance report
func (m *MetricsCollector) GenerateReport() *PerformanceReport {
	report := &PerformanceReport{
		Timestamp: time.Now(),
		Summary:   m.generateSummary(),
		Results:   m.results,
	}

	return report
}

// SaveReport saves the report to files (JSON and text)
func (m *MetricsCollector) SaveReport(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	report := m.GenerateReport()

	// Save JSON report
	jsonFile := fmt.Sprintf("%s/performance_report_%s.json", outputDir, time.Now().Format("20060102_150405"))
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON report: %w", err)
	}

	// Save text summary
	textFile := fmt.Sprintf("%s/performance_summary_%s.txt", outputDir, time.Now().Format("20060102_150405"))
	textData := m.generateTextSummary()
	if err := os.WriteFile(textFile, []byte(textData), 0644); err != nil {
		return fmt.Errorf("failed to write text report: %w", err)
	}

	fmt.Printf("Reports saved to:\n  JSON: %s\n  Text: %s\n", jsonFile, textFile)
	return nil
}

// generateSummary creates performance summary statistics
func (m *MetricsCollector) generateSummary() map[string]OperationSummary {
	summary := make(map[string]OperationSummary)

	// Group results by operation
	operationGroups := make(map[string][]BenchmarkResult)
	for _, result := range m.results {
		key := fmt.Sprintf("%s_%s", result.Operation, result.Scenario)
		operationGroups[key] = append(operationGroups[key], result)
	}

	// Calculate statistics for each operation
	for operation, results := range operationGroups {
		if len(results) == 0 {
			continue
		}

		clientStats := make(map[string]ClientStats)

		// Group by client
		clientGroups := make(map[string][]BenchmarkResult)
		for _, result := range results {
			clientGroups[result.Client] = append(clientGroups[result.Client], result)
		}

		// Calculate stats per client
		for client, clientResults := range clientGroups {
			durations := make([]time.Duration, 0, len(clientResults))
			memories := make([]int64, 0, len(clientResults))
			successCount := 0

			for _, result := range clientResults {
				durations = append(durations, result.Duration)
				memories = append(memories, result.MemoryUsed)
				if result.Success {
					successCount++
				}
			}

			sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
			sort.Slice(memories, func(i, j int) bool { return memories[i] < memories[j] })

			stats := ClientStats{
				Count:          len(clientResults),
				SuccessRate:    float64(successCount) / float64(len(clientResults)),
				AvgDuration:    calculateAverage(durations),
				MedianDuration: calculateMedian(durations),
				P95Duration:    calculatePercentile(durations, 0.95),
				AvgMemory:      calculateAverageMemory(memories),
				MedianMemory:   calculateMedianMemory(memories),
			}

			clientStats[client] = stats
		}

		summary[operation] = OperationSummary{
			Operation:   operation,
			ClientStats: clientStats,
		}
	}

	return summary
}

// generateTextSummary creates a human-readable text summary
func (m *MetricsCollector) generateTextSummary() string {
	summary := m.generateSummary()

	text := fmt.Sprintf("Performance Benchmark Report\n")
	text += fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339))
	text += fmt.Sprintf("Total Benchmarks: %d\n\n", len(m.results))

	// Add comparison tables at the top
	text += m.generateComparisonTables(summary)

	// Sort operations by name
	operations := make([]string, 0, len(summary))
	for operation := range summary {
		operations = append(operations, operation)
	}
	sort.Strings(operations)

	text += "=== DETAILED RESULTS ===\n\n"

	for _, operation := range operations {
		opSummary := summary[operation]
		text += fmt.Sprintf("=== %s ===\n", operation)

		// Sort clients by name
		clients := make([]string, 0, len(opSummary.ClientStats))
		for client := range opSummary.ClientStats {
			clients = append(clients, client)
		}
		sort.Strings(clients)

		for _, client := range clients {
			stats := opSummary.ClientStats[client]
			text += fmt.Sprintf("\n%s:\n", client)
			text += fmt.Sprintf("  Runs: %d\n", stats.Count)
			text += fmt.Sprintf("  Success Rate: %.2f%%\n", stats.SuccessRate*100)
			text += fmt.Sprintf("  Duration - Avg: %v, Median: %v, P95: %v\n",
				stats.AvgDuration, stats.MedianDuration, stats.P95Duration)
			text += fmt.Sprintf("  Memory - Avg: %s, Median: %s\n",
				formatBytes(stats.AvgMemory), formatBytes(stats.MedianMemory))
		}
		text += "\n"
	}

	return text
}

// generateComparisonTables creates comparison tables for all clients
func (m *MetricsCollector) generateComparisonTables(summary map[string]OperationSummary) string {
	text := "=== PERFORMANCE COMPARISON TABLES ===\n\n"
	
	// Get all operations and clients
	operations := make([]string, 0, len(summary))
	clientSet := make(map[string]bool)
	
	for operation, opSummary := range summary {
		operations = append(operations, operation)
		for client := range opSummary.ClientStats {
			clientSet[client] = true
		}
	}
	
	sort.Strings(operations)
	clients := make([]string, 0, len(clientSet))
	for client := range clientSet {
		clients = append(clients, client)
	}
	sort.Strings(clients)
	
	// Table 1: Average Duration Comparison
	text += "1. AVERAGE DURATION COMPARISON\n"
	text += m.generateDurationTable(summary, operations, clients)
	text += "\n"
	
	// Table 2: Average Memory Comparison  
	text += "2. AVERAGE MEMORY USAGE COMPARISON\n"
	text += m.generateMemoryTable(summary, operations, clients)
	text += "\n"
	
	// Table 3: Nanogit Performance vs Others (Duration)
	text += "3. NANOGIT DURATION PERFORMANCE vs OTHERS\n"
	text += "(Shows how many times faster/slower nanogit is)\n"
	text += m.generateNanogitComparisonTable(summary, operations, clients, "duration")
	text += "\n"
	
	// Table 4: Nanogit Performance vs Others (Memory)
	text += "4. NANOGIT MEMORY USAGE vs OTHERS\n"
	text += "(Shows how many times less/more memory nanogit uses)\n"
	text += m.generateNanogitComparisonTable(summary, operations, clients, "memory")
	text += "\n"
	
	return text
}

// generateDurationTable creates a duration comparison table
func (m *MetricsCollector) generateDurationTable(summary map[string]OperationSummary, operations, clients []string) string {
	// Calculate column widths
	maxOpWidth := 20
	for _, op := range operations {
		if len(op) > maxOpWidth {
			maxOpWidth = len(op)
		}
	}
	
	colWidth := 12
	
	// Header
	text := fmt.Sprintf("%-*s", maxOpWidth, "Scenario")
	for _, client := range clients {
		text += fmt.Sprintf(" %*s", colWidth, client)
	}
	text += fmt.Sprintf(" %*s", colWidth, "Best")
	text += "\n"
	
	// Separator
	text += fmt.Sprintf("%s", strings.Repeat("-", maxOpWidth))
	for range clients {
		text += fmt.Sprintf(" %s", strings.Repeat("-", colWidth))
	}
	text += fmt.Sprintf(" %s", strings.Repeat("-", colWidth))
	text += "\n"
	
	// Data rows
	for _, operation := range operations {
		opSummary, exists := summary[operation]
		if !exists {
			continue
		}
		
		text += fmt.Sprintf("%-*s", maxOpWidth, operation)
		
		bestDuration := time.Duration(0)
		bestClient := ""
		durations := make(map[string]time.Duration)
		
		for _, client := range clients {
			if stats, exists := opSummary.ClientStats[client]; exists {
				duration := stats.AvgDuration
				durations[client] = duration
				text += fmt.Sprintf(" %*s", colWidth, formatDuration(duration))
				
				if bestDuration == 0 || duration < bestDuration {
					bestDuration = duration
					bestClient = client
				}
			} else {
				text += fmt.Sprintf(" %*s", colWidth, "N/A")
			}
		}
		
		text += fmt.Sprintf(" %*s", colWidth, bestClient)
		text += "\n"
	}
	
	return text
}

// generateMemoryTable creates a memory comparison table
func (m *MetricsCollector) generateMemoryTable(summary map[string]OperationSummary, operations, clients []string) string {
	// Calculate column widths
	maxOpWidth := 20
	for _, op := range operations {
		if len(op) > maxOpWidth {
			maxOpWidth = len(op)
		}
	}
	
	colWidth := 12
	
	// Header
	text := fmt.Sprintf("%-*s", maxOpWidth, "Scenario")
	for _, client := range clients {
		text += fmt.Sprintf(" %*s", colWidth, client)
	}
	text += fmt.Sprintf(" %*s", colWidth, "Best")
	text += "\n"
	
	// Separator
	text += fmt.Sprintf("%s", strings.Repeat("-", maxOpWidth))
	for range clients {
		text += fmt.Sprintf(" %s", strings.Repeat("-", colWidth))
	}
	text += fmt.Sprintf(" %s", strings.Repeat("-", colWidth))
	text += "\n"
	
	// Data rows
	for _, operation := range operations {
		opSummary, exists := summary[operation]
		if !exists {
			continue
		}
		
		text += fmt.Sprintf("%-*s", maxOpWidth, operation)
		
		bestMemory := int64(0)
		bestClient := ""
		
		for _, client := range clients {
			if stats, exists := opSummary.ClientStats[client]; exists {
				memory := stats.AvgMemory
				text += fmt.Sprintf(" %*s", colWidth, formatBytes(memory))
				
				if bestMemory == 0 || memory < bestMemory {
					bestMemory = memory
					bestClient = client
				}
			} else {
				text += fmt.Sprintf(" %*s", colWidth, "N/A")
			}
		}
		
		text += fmt.Sprintf(" %*s", colWidth, bestClient)
		text += "\n"
	}
	
	return text
}

// generateNanogitComparisonTable creates a comparison table showing nanogit performance vs others
func (m *MetricsCollector) generateNanogitComparisonTable(summary map[string]OperationSummary, operations, clients []string, metric string) string {
	// Calculate column widths
	maxOpWidth := 20
	for _, op := range operations {
		if len(op) > maxOpWidth {
			maxOpWidth = len(op)
		}
	}
	
	colWidth := 12
	
	// Header
	text := fmt.Sprintf("%-*s", maxOpWidth, "Scenario")
	for _, client := range clients {
		if client != "nanogit" {
			text += fmt.Sprintf(" %*s", colWidth, "vs "+client)
		}
	}
	text += "\n"
	
	// Separator
	text += fmt.Sprintf("%s", strings.Repeat("-", maxOpWidth))
	for _, client := range clients {
		if client != "nanogit" {
			text += fmt.Sprintf(" %s", strings.Repeat("-", colWidth))
		}
	}
	text += "\n"
	
	// Data rows
	for _, operation := range operations {
		opSummary, exists := summary[operation]
		if !exists {
			continue
		}
		
		nanogitStats, nanogitExists := opSummary.ClientStats["nanogit"]
		if !nanogitExists {
			continue
		}
		
		text += fmt.Sprintf("%-*s", maxOpWidth, operation)
		
		for _, client := range clients {
			if client == "nanogit" {
				continue
			}
			
			if stats, exists := opSummary.ClientStats[client]; exists {
				var nanogitValue, otherValue float64
				
				if metric == "duration" {
					nanogitValue = float64(nanogitStats.AvgDuration.Nanoseconds())
					otherValue = float64(stats.AvgDuration.Nanoseconds())
				} else { // memory
					nanogitValue = float64(nanogitStats.AvgMemory)
					otherValue = float64(stats.AvgMemory)
				}
				
				var multiplier float64
				var displayText string
				
				if nanogitValue != 0 && otherValue != 0 {
					if metric == "duration" {
						// For duration: how many times faster is nanogit
						multiplier = otherValue / nanogitValue
						if multiplier >= 1.0 {
							displayText = fmt.Sprintf("%.1fx faster", multiplier)
						} else {
							// nanogit is slower
							multiplier = nanogitValue / otherValue
							displayText = fmt.Sprintf("%.1fx slower", multiplier)
						}
					} else { // memory
						// For memory: how many times less memory does nanogit use
						if nanogitValue < otherValue {
							multiplier = otherValue / nanogitValue
							displayText = fmt.Sprintf("%.1fx less", multiplier)
						} else {
							// nanogit uses more memory
							multiplier = nanogitValue / otherValue
							displayText = fmt.Sprintf("%.1fx more", multiplier)
						}
					}
				} else {
					displayText = "N/A"
				}
				
				text += fmt.Sprintf(" %*s", colWidth, displayText)
			} else {
				text += fmt.Sprintf(" %*s", colWidth, "N/A")
			}
		}
		
		text += "\n"
	}
	
	return text
}

// formatDuration formats duration for table display
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// Helper functions for statistical calculations
func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func calculateMedian(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	mid := len(durations) / 2
	if len(durations)%2 == 0 {
		return (durations[mid-1] + durations[mid]) / 2
	}
	return durations[mid]
}

func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	index := int(float64(len(durations)) * percentile)
	if index >= len(durations) {
		index = len(durations) - 1
	}
	return durations[index]
}

func calculateAverageMemory(memories []int64) int64 {
	if len(memories) == 0 {
		return 0
	}
	var total int64
	for _, m := range memories {
		total += m
	}
	return total / int64(len(memories))
}

func calculateMedianMemory(memories []int64) int64 {
	if len(memories) == 0 {
		return 0
	}
	mid := len(memories) / 2
	if len(memories)%2 == 0 {
		return (memories[mid-1] + memories[mid]) / 2
	}
	return memories[mid]
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Report structures
type PerformanceReport struct {
	Timestamp time.Time                   `json:"timestamp"`
	Summary   map[string]OperationSummary `json:"summary"`
	Results   []BenchmarkResult           `json:"results"`
}

type OperationSummary struct {
	Operation   string                 `json:"operation"`
	ClientStats map[string]ClientStats `json:"client_stats"`
}

type ClientStats struct {
	Count          int           `json:"count"`
	SuccessRate    float64       `json:"success_rate"`
	AvgDuration    time.Duration `json:"avg_duration"`
	MedianDuration time.Duration `json:"median_duration"`
	P95Duration    time.Duration `json:"p95_duration"`
	AvgMemory      int64         `json:"avg_memory"`
	MedianMemory   int64         `json:"median_memory"`
}
