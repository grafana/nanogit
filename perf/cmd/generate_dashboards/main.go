package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// DashboardSpec defines the characteristics of a Grafana dashboard
type DashboardSpec struct {
	Name           string
	PanelCount     int
	TemplateVars   int
	SizeCategory   string
	TargetSizeKB   int
	HasAnnotations bool
	HasAlerts      bool
	DataSources    []string
}

// GetDashboardSpecs returns predefined dashboard specifications
func GetDashboardSpecs() []DashboardSpec {
	return []DashboardSpec{
		{
			Name:           "small",
			PanelCount:     8,
			TemplateVars:   3,
			SizeCategory:   "small",
			TargetSizeKB:   15,
			HasAnnotations: false,
			HasAlerts:      false,
			DataSources:    []string{"prometheus"},
		},
		{
			Name:           "medium",
			PanelCount:     35,
			TemplateVars:   12,
			SizeCategory:   "medium",
			TargetSizeKB:   75,
			HasAnnotations: true,
			HasAlerts:      true,
			DataSources:    []string{"prometheus", "loki", "tempo"},
		},
		{
			Name:           "large",
			PanelCount:     85,
			TemplateVars:   20,
			SizeCategory:   "large",
			TargetSizeKB:   300,
			HasAnnotations: true,
			HasAlerts:      true,
			DataSources:    []string{"prometheus", "loki", "tempo", "elasticsearch", "mysql"},
		},
		{
			Name:           "xlarge",
			PanelCount:     220,
			TemplateVars:   35,
			SizeCategory:   "xlarge",
			TargetSizeKB:   1200,
			HasAnnotations: true,
			HasAlerts:      true,
			DataSources:    []string{"prometheus", "loki", "tempo", "elasticsearch", "mysql", "postgres", "influxdb", "cloudwatch"},
		},
	}
}

// GrafanaDashboard represents a Grafana dashboard JSON structure
type GrafanaDashboard struct {
	ID              int                    `json:"id"`
	Title           string                 `json:"title"`
	Tags            []string               `json:"tags"`
	Style           string                 `json:"style"`
	Timezone        string                 `json:"timezone"`
	Panels          []Panel                `json:"panels"`
	Templating      Templating             `json:"templating"`
	Time            TimeRange              `json:"time"`
	Timepicker      interface{}            `json:"timepicker"`
	Refresh         string                 `json:"refresh"`
	SchemaVersion   int                    `json:"schemaVersion"`
	Version         int                    `json:"version"`
	Links           []interface{}          `json:"links"`
	Annotations     Annotations            `json:"annotations"`
	Editable        bool                   `json:"editable"`
	FiscalYearStartMonth int               `json:"fiscalYearStartMonth"`
	GraphTooltip    int                    `json:"graphTooltip"`
	HideControls    bool                   `json:"hideControls"`
	LiveNow         bool                   `json:"liveNow"`
	WeekStart       string                 `json:"weekStart"`
}

type Panel struct {
	ID              int                    `json:"id"`
	Title           string                 `json:"title"`
	Type            string                 `json:"type"`
	GridPos         GridPos                `json:"gridPos"`
	Targets         []Target               `json:"targets"`
	FieldConfig     FieldConfig            `json:"fieldConfig"`
	Options         map[string]interface{} `json:"options"`
	Transparent     bool                   `json:"transparent"`
	Datasource      Datasource             `json:"datasource"`
	PluginVersion   string                 `json:"pluginVersion"`
	Description     string                 `json:"description"`
	Links           []interface{}          `json:"links"`
	Repeat          string                 `json:"repeat,omitempty"`
	RepeatDirection string                 `json:"repeatDirection,omitempty"`
	MaxDataPoints   int                    `json:"maxDataPoints,omitempty"`
	Interval        string                 `json:"interval,omitempty"`
	Thresholds      []interface{}          `json:"thresholds,omitempty"`
	Alert           *Alert                 `json:"alert,omitempty"`
}

type GridPos struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

type Target struct {
	Expr           string                 `json:"expr"`
	RefID          string                 `json:"refId"`
	LegendFormat   string                 `json:"legendFormat"`
	Format         string                 `json:"format"`
	Interval       string                 `json:"interval"`
	IntervalFactor int                    `json:"intervalFactor"`
	Step           int                    `json:"step"`
	Hide           bool                   `json:"hide"`
	Datasource     Datasource             `json:"datasource"`
	ExtraData      map[string]interface{} `json:"extraData,omitempty"`
}

type Datasource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type FieldConfig struct {
	Defaults  FieldDefaults           `json:"defaults"`
	Overrides []map[string]interface{} `json:"overrides"`
}

type FieldDefaults struct {
	Color         map[string]interface{} `json:"color"`
	Custom        map[string]interface{} `json:"custom"`
	Mappings      []interface{}          `json:"mappings"`
	Thresholds    Thresholds             `json:"thresholds"`
	Unit          string                 `json:"unit"`
	Min           *float64               `json:"min,omitempty"`
	Max           *float64               `json:"max,omitempty"`
	Decimals      *int                   `json:"decimals,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	NoValue       string                 `json:"noValue,omitempty"`
	Description   string                 `json:"description,omitempty"`
}

type Thresholds struct {
	Mode  string      `json:"mode"`
	Steps []Threshold `json:"steps"`
}

type Threshold struct {
	Color string   `json:"color"`
	Value *float64 `json:"value"`
}

type Templating struct {
	List []TemplateVar `json:"list"`
}

type TemplateVar struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Query       string                 `json:"query"`
	Current     map[string]interface{} `json:"current"`
	Options     []interface{}          `json:"options"`
	Refresh     int                    `json:"refresh"`
	Regex       string                 `json:"regex"`
	Sort        int                    `json:"sort"`
	Multi       bool                   `json:"multi"`
	IncludeAll  bool                   `json:"includeAll"`
	AllValue    string                 `json:"allValue"`
	Hide        int                    `json:"hide"`
	Datasource  *Datasource            `json:"datasource,omitempty"`
}

type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Annotations struct {
	List []Annotation `json:"list"`
}

type Annotation struct {
	Name       string                 `json:"name"`
	Enable     bool                   `json:"enable"`
	Hide       bool                   `json:"hide"`
	IconColor  string                 `json:"iconColor"`
	Query      string                 `json:"query"`
	ShowLine   bool                   `json:"showLine"`
	LineColor  string                 `json:"lineColor"`
	TextFormat string                 `json:"textFormat"`
	Datasource Datasource             `json:"datasource"`
	Target     map[string]interface{} `json:"target"`
}

type Alert struct {
	Name            string                   `json:"name"`
	Message         string                   `json:"message"`
	Frequency       string                   `json:"frequency"`
	Conditions      []map[string]interface{} `json:"conditions"`
	ExecutionErrorState string               `json:"executionErrorState"`
	NoDataState     string                   `json:"noDataState"`
	For             string                   `json:"for"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Create output directory
	outputDir := "./generated_dashboards"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	specs := GetDashboardSpecs()

	for _, spec := range specs {
		fmt.Printf("Generating %s dashboard (%d panels, target: %dKB)...\n", 
			spec.Name, spec.PanelCount, spec.TargetSizeKB)

		dashboard := generateDashboard(spec)
		
		// Save dashboard
		filename := fmt.Sprintf("%s-dashboard.json", spec.Name)
		filepath := filepath.Join(outputDir, filename)
		
		if err := saveDashboard(dashboard, filepath); err != nil {
			log.Fatalf("Failed to save %s dashboard: %v", spec.Name, err)
		}

		// Check file size
		if stat, err := os.Stat(filepath); err == nil {
			sizeKB := stat.Size() / 1024
			fmt.Printf("Created %s (%dKB)\n", filepath, sizeKB)
		}
	}

	fmt.Println("Dashboard generation complete!")
}

func generateDashboard(spec DashboardSpec) GrafanaDashboard {
	dashboard := GrafanaDashboard{
		ID:              rand.Intn(1000000),
		Title:           fmt.Sprintf("%s Dashboard - %s", capitalizeFirst(spec.Name), generateTitle()),
		Tags:            generateTags(spec),
		Style:           "dark",
		Timezone:        "browser",
		Panels:          generatePanels(spec),
		Templating:      generateTemplating(spec),
		Time:            TimeRange{From: "now-1h", To: "now"},
		Timepicker:      map[string]interface{}{},
		Refresh:         "30s",
		SchemaVersion:   30,
		Version:         rand.Intn(100) + 1,
		Links:           []interface{}{},
		Annotations:     generateAnnotations(spec),
		Editable:        true,
		FiscalYearStartMonth: 0,
		GraphTooltip:    0,
		HideControls:    false,
		LiveNow:         false,
		WeekStart:       "",
	}

	return dashboard
}

func generatePanels(spec DashboardSpec) []Panel {
	panels := make([]Panel, spec.PanelCount)
	
	x, y := 0, 0
	panelHeight := 8
	
	for i := 0; i < spec.PanelCount; i++ {
		panelType := choosePanelType(spec)
		
		// Adjust panel size based on type
		w, h := getPanelSize(panelType, spec.SizeCategory)
		
		// Wrap to next row if needed
		if x+w > 24 {
			x = 0
			y += panelHeight
		}
		
		panels[i] = Panel{
			ID:    i + 1,
			Title: generatePanelTitle(panelType, i),
			Type:  panelType,
			GridPos: GridPos{
				H: h,
				W: w,
				X: x,
				Y: y,
			},
			Targets:       generateTargets(spec, panelType),
			FieldConfig:   generateFieldConfig(panelType),
			Options:       generatePanelOptions(panelType, spec),
			Transparent:   rand.Float32() < 0.1,
			Datasource:    chooseDatasource(spec.DataSources),
			PluginVersion: "8.5.0",
			Description:   generatePanelDescription(panelType),
			Links:         []interface{}{},
			MaxDataPoints: 300,
			Interval:      "1m",
		}
		
		// Add alert if specified
		if spec.HasAlerts && rand.Float32() < 0.15 {
			panels[i].Alert = generateAlert(panelType)
		}
		
		x += w
	}
	
	return panels
}

func choosePanelType(spec DashboardSpec) string {
	types := []string{"timeseries", "stat", "gauge", "table", "heatmap", "piechart", "bargauge", "text"}
	
	// Weight towards common types
	weights := map[string]float32{
		"timeseries": 0.4,
		"stat":       0.2,
		"gauge":      0.1,
		"table":      0.1,
		"heatmap":    0.05,
		"piechart":   0.05,
		"bargauge":   0.05,
		"text":       0.05,
	}
	
	r := rand.Float32()
	cumulative := float32(0)
	
	for _, panelType := range types {
		cumulative += weights[panelType]
		if r <= cumulative {
			return panelType
		}
	}
	
	return "timeseries"
}

func getPanelSize(panelType, sizeCategory string) (int, int) {
	baseSizes := map[string][2]int{
		"timeseries": {12, 8},
		"stat":       {6, 4},
		"gauge":      {6, 6},
		"table":      {24, 8},
		"heatmap":    {12, 8},
		"piechart":   {8, 8},
		"bargauge":   {6, 6},
		"text":       {12, 4},
	}
	
	size := baseSizes[panelType]
	w, h := size[0], size[1]
	
	// Adjust for dashboard size category
	switch sizeCategory {
	case "xlarge":
		if rand.Float32() < 0.3 {
			w = min(w+6, 24)
			h = min(h+2, 12)
		}
	case "large":
		if rand.Float32() < 0.2 {
			w = min(w+3, 24)
			h = min(h+1, 10)
		}
	}
	
	return w, h
}

func generateTargets(spec DashboardSpec, panelType string) []Target {
	targetCount := 1
	if spec.SizeCategory == "large" || spec.SizeCategory == "xlarge" {
		targetCount = rand.Intn(3) + 1
	}
	
	targets := make([]Target, targetCount)
	
	for i := 0; i < targetCount; i++ {
		targets[i] = Target{
			Expr:           generateQuery(spec.DataSources[rand.Intn(len(spec.DataSources))], panelType),
			RefID:          string(rune('A' + i)),
			LegendFormat:   generateLegendFormat(),
			Format:         "time_series",
			Interval:       "1m",
			IntervalFactor: 1,
			Step:           60,
			Hide:           false,
			Datasource:     chooseDatasource(spec.DataSources),
		}
		
		// Add extra complexity for larger dashboards
		if spec.SizeCategory == "xlarge" {
			targets[i].ExtraData = map[string]interface{}{
				"exemplar":     true,
				"instant":      false,
				"range":        true,
				"resolution":   1,
				"maxDataPoints": 43200,
			}
		}
	}
	
	return targets
}

func generateQuery(datasourceType, panelType string) string {
	switch datasourceType {
	case "prometheus":
		metrics := []string{"cpu_usage", "memory_usage", "disk_usage", "network_io", "http_requests_total", "response_time"}
		metric := metrics[rand.Intn(len(metrics))]
		return fmt.Sprintf("rate(%s[5m])", metric)
	case "loki":
		return `{job="app"} |= "error" | json | rate[5m]`
	case "tempo":
		return `{service.name="frontend"}`
	case "elasticsearch":
		return `{"query": {"match_all": {}}}`
	default:
		return "up"
	}
}

func generateLegendFormat() string {
	formats := []string{"{{instance}}", "{{job}}", "{{service}}", "{{environment}}", "Series {{refId}}"}
	return formats[rand.Intn(len(formats))]
}

func chooseDatasource(datasources []string) Datasource {
	ds := datasources[rand.Intn(len(datasources))]
	return Datasource{
		Type: ds,
		UID:  fmt.Sprintf("%s-uid-%d", ds, rand.Intn(1000)),
		Name: fmt.Sprintf("%s-datasource", ds),
	}
}

func generateFieldConfig(panelType string) FieldConfig {
	return FieldConfig{
		Defaults: FieldDefaults{
			Color: map[string]interface{}{
				"mode": "palette-classic",
			},
			Custom: generateCustomConfig(panelType),
			Mappings: []interface{}{},
			Thresholds: Thresholds{
				Mode: "absolute",
				Steps: []Threshold{
					{Color: "green", Value: nil},
					{Color: "red", Value: float64Ptr(80)},
				},
			},
			Unit: chooseUnit(panelType),
		},
		Overrides: []map[string]interface{}{},
	}
}

func generateCustomConfig(panelType string) map[string]interface{} {
	switch panelType {
	case "timeseries":
		return map[string]interface{}{
			"drawStyle":         "line",
			"lineInterpolation": "linear",
			"lineWidth":         1,
			"fillOpacity":       0,
			"gradientMode":      "none",
			"spanNulls":         false,
			"insertNulls":       false,
			"showPoints":        "auto",
			"pointSize":         5,
			"stacking": map[string]interface{}{
				"mode":  "none",
				"group": "A",
			},
			"axisPlacement": "auto",
			"axisLabel":     "",
			"scaleDistribution": map[string]interface{}{
				"type": "linear",
			},
			"hideFrom": map[string]interface{}{
				"legend":  false,
				"tooltip": false,
				"vis":     false,
			},
			"thresholdsStyle": map[string]interface{}{
				"mode": "off",
			},
		}
	case "stat":
		return map[string]interface{}{
			"orientation":    "auto",
			"reduceOptions": map[string]interface{}{
				"values": false,
				"calcs":  []string{"lastNotNull"},
				"fields": "",
			},
			"textMode":    "auto",
			"colorMode":   "value",
			"graphMode":   "area",
			"justifyMode": "auto",
		}
	default:
		return map[string]interface{}{}
	}
}

func chooseUnit(panelType string) string {
	units := []string{"short", "percent", "bytes", "ms", "ops", "reqps", "none"}
	return units[rand.Intn(len(units))]
}

func generatePanelOptions(panelType string, spec DashboardSpec) map[string]interface{} {
	options := map[string]interface{}{}
	
	switch panelType {
	case "timeseries":
		options["tooltip"] = map[string]interface{}{
			"mode": "single",
			"sort": "none",
		}
		options["legend"] = map[string]interface{}{
			"displayMode": "visible",
			"placement":   "bottom",
			"calcs":       []string{},
		}
	case "table":
		options["showHeader"] = true
		options["sortBy"] = []map[string]interface{}{
			{"desc": true, "displayName": "Value"},
		}
	case "gauge":
		options["reduceOptions"] = map[string]interface{}{
			"values": false,
			"calcs":  []string{"lastNotNull"},
			"fields": "",
		}
		options["orientation"] = "auto"
		options["showThresholdLabels"] = false
		options["showThresholdMarkers"] = true
	}
	
	// Add more complexity for larger dashboards
	if spec.SizeCategory == "xlarge" {
		options["displayMode"] = "table"
		options["placement"] = "right"
		options["showLegend"] = true
		options["sortBy"] = []map[string]interface{}{
			{"desc": false, "displayName": "Time"},
		}
	}
	
	return options
}

func generatePanelTitle(panelType string, index int) string {
	titlePrefixes := map[string][]string{
		"timeseries": {"CPU Usage", "Memory Usage", "Network Traffic", "Response Time", "Request Rate", "Error Rate"},
		"stat":       {"Total Requests", "Active Users", "System Load", "Uptime", "Success Rate", "Cache Hit Rate"},
		"gauge":      {"CPU Load", "Memory Usage", "Disk Usage", "Network Utilization", "Queue Size", "Thread Count"},
		"table":      {"Recent Events", "Top Errors", "Service Status", "Resource Usage", "Performance Metrics", "Alert Summary"},
		"heatmap":    {"Response Time Distribution", "Request Volume Heatmap", "Error Rate Heatmap", "Usage Patterns", "Performance Matrix", "Load Distribution"},
		"piechart":   {"Error Distribution", "Service Breakdown", "Resource Allocation", "Request Types", "User Segments", "Status Distribution"},
		"bargauge":   {"Service Performance", "Resource Utilization", "Team Metrics", "SLA Compliance", "Quality Metrics", "Capacity Usage"},
		"text":       {"System Overview", "Important Notes", "Troubleshooting Guide", "Service Information", "Alert Instructions", "Contact Information"},
	}
	
	titles := titlePrefixes[panelType]
	if titles == nil {
		return fmt.Sprintf("Panel %d", index+1)
	}
	
	return titles[rand.Intn(len(titles))]
}

func generatePanelDescription(panelType string) string {
	descriptions := map[string][]string{
		"timeseries": {"Shows the trend over time", "Displays historical data patterns", "Monitors real-time metrics"},
		"stat":       {"Current value indicator", "Single value metric display", "Key performance indicator"},
		"gauge":      {"Visual progress indicator", "Shows current level vs threshold", "Capacity utilization display"},
		"table":      {"Detailed data breakdown", "Structured information display", "Comprehensive data view"},
		"heatmap":    {"Distribution visualization", "Pattern identification display", "Intensity mapping"},
		"piechart":   {"Proportion visualization", "Category breakdown display", "Percentage distribution"},
		"bargauge":   {"Comparative values display", "Multi-metric comparison", "Performance comparison"},
		"text":       {"Informational content", "Documentation panel", "Instructional text"},
	}
	
	desc := descriptions[panelType]
	if desc == nil {
		return "Generated panel description"
	}
	
	return desc[rand.Intn(len(desc))]
}

func generateAlert(panelType string) *Alert {
	return &Alert{
		Name:            fmt.Sprintf("Alert for %s panel", panelType),
		Message:         "Alert condition met",
		Frequency:       "10s",
		Conditions:      []map[string]interface{}{},
		ExecutionErrorState: "alerting",
		NoDataState:     "no_data",
		For:             "5m",
	}
}

func generateTemplating(spec DashboardSpec) Templating {
	vars := make([]TemplateVar, spec.TemplateVars)
	
	varNames := []string{"environment", "service", "instance", "job", "region", "cluster", "namespace", "pod", "container", "node"}
	
	for i := 0; i < spec.TemplateVars; i++ {
		name := varNames[i%len(varNames)]
		if i >= len(varNames) {
			name = fmt.Sprintf("%s_%d", name, i/len(varNames))
		}
		
		vars[i] = TemplateVar{
			Name:        name,
			Type:        "query",
			Label:       capitalizeFirst(name),
			Description: fmt.Sprintf("Select %s", name),
			Query:       fmt.Sprintf("label_values(%s)", name),
			Current: map[string]interface{}{
				"selected": true,
				"text":     "All",
				"value":    "$__all",
			},
			Options:     []interface{}{},
			Refresh:     1,
			Regex:       "",
			Sort:        1,
			Multi:       true,
			IncludeAll:  true,
			AllValue:    "",
			Hide:        0,
		}
	}
	
	return Templating{List: vars}
}

func generateAnnotations(spec DashboardSpec) Annotations {
	if !spec.HasAnnotations {
		return Annotations{List: []Annotation{}}
	}
	
	annotations := []Annotation{
		{
			Name:       "Deployments",
			Enable:     true,
			Hide:       false,
			IconColor:  "rgba(0, 211, 255, 1)",
			Query:      "deployment_events",
			ShowLine:   true,
			LineColor:  "rgba(0, 211, 255, 1)",
			TextFormat: "{{title}}: {{text}}",
			Datasource: chooseDatasource(spec.DataSources),
			Target:     map[string]interface{}{},
		},
	}
	
	return Annotations{List: annotations}
}

func generateTags(spec DashboardSpec) []string {
	allTags := []string{"monitoring", "infrastructure", "application", "performance", "alerts", "business", "devops", "sre", "kubernetes", "docker"}
	
	tagCount := min(5, len(allTags))
	if spec.SizeCategory == "xlarge" {
		tagCount = min(8, len(allTags))
	}
	
	selectedTags := make([]string, tagCount)
	for i := 0; i < tagCount; i++ {
		selectedTags[i] = allTags[rand.Intn(len(allTags))]
	}
	
	return selectedTags
}

func generateTitle() string {
	adjectives := []string{"Production", "Staging", "Development", "Critical", "Essential", "Primary", "Secondary", "Advanced", "Basic", "Comprehensive"}
	nouns := []string{"Monitoring", "Metrics", "Performance", "Overview", "Analytics", "Insights", "Operations", "Health", "Status", "Report"}
	
	return fmt.Sprintf("%s %s", adjectives[rand.Intn(len(adjectives))], nouns[rand.Intn(len(nouns))])
}

func saveDashboard(dashboard GrafanaDashboard, filepath string) error {
	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard: %w", err)
	}
	
	return os.WriteFile(filepath, data, 0644)
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return fmt.Sprintf("%c%s", s[0]-32, s[1:])
}

func float64Ptr(f float64) *float64 {
	return &f
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}