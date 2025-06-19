package performance

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
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
				Count:        len(clientResults),
				SuccessRate:  float64(successCount) / float64(len(clientResults)),
				AvgDuration:  calculateAverage(durations),
				MedianDuration: calculateMedian(durations),
				P95Duration:  calculatePercentile(durations, 0.95),
				AvgMemory:    calculateAverageMemory(memories),
				MedianMemory: calculateMedianMemory(memories),
			}
			
			clientStats[client] = stats
		}
		
		summary[operation] = OperationSummary{
			Operation:    operation,
			ClientStats:  clientStats,
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
	
	// Sort operations by name
	operations := make([]string, 0, len(summary))
	for operation := range summary {
		operations = append(operations, operation)
	}
	sort.Strings(operations)
	
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
	Timestamp time.Time                    `json:"timestamp"`
	Summary   map[string]OperationSummary  `json:"summary"`
	Results   []BenchmarkResult            `json:"results"`
}

type OperationSummary struct {
	Operation   string                  `json:"operation"`
	ClientStats map[string]ClientStats  `json:"client_stats"`
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