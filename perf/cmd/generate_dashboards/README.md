# Grafana Dashboard Generator

A utility to generate realistic Grafana dashboards of various sizes for performance testing and development purposes.

## Overview

This tool creates valid Grafana dashboard JSON files that mimic real-world dashboard complexity and structure. It's designed to help test scenarios involving large dashboards, Git repository performance with varying file sizes, and Grafana import/export operations.

## Usage

```bash
cd perf/cmd/generate_dashboards
go run main.go
```

The tool will create a `generated_dashboards/` directory containing four dashboard files:

- `small-dashboard.json` (~18KB)
- `medium-dashboard.json` (~54KB) 
- `large-dashboard.json` (~144KB)
- `xlarge-dashboard.json` (~462KB)

## Dashboard Size Categories

### Small Dashboard (~15-25KB target)
- **8 panels** - Basic monitoring setup
- **3 template variables** - Simple filtering
- **Single datasource** - Prometheus only
- **Use case**: Small team dashboard, service overview
- **Real-world equivalent**: Basic application monitoring, simple infrastructure overview

### Medium Dashboard (~50-100KB target)
- **25 panels** - Comprehensive monitoring
- **8 template variables** - Multi-dimensional filtering
- **3 datasources** - Prometheus, Loki, Tempo
- **Annotations enabled** - Deployment tracking
- **Basic alerts** - Error rate monitoring
- **Use case**: Department dashboard, multi-service monitoring
- **Real-world equivalent**: Full-stack application monitoring, team KPI dashboard

### Large Dashboard (~200-400KB target)
- **60 panels** - Enterprise-scale monitoring
- **15 template variables** - Complex filtering and grouping
- **5 datasources** - Multiple observability tools
- **Advanced features** - Complex queries, custom visualizations
- **Alert rules** - Comprehensive alerting
- **Use case**: Enterprise service monitoring, multi-team dashboard
- **Real-world equivalent**: Platform monitoring, business intelligence dashboard

### XLarge Dashboard (~800KB-1.5MB target)
- **150+ panels** - Massive monitoring setup
- **25+ template variables** - Extensive filtering capabilities
- **8+ datasources** - Full observability stack
- **Complex configurations** - Advanced panel options, overrides
- **Extensive metadata** - Rich descriptions, links, annotations
- **Use case**: Global enterprise monitoring, multi-region dashboards
- **Real-world equivalent**: Corporate-wide monitoring, compliance dashboards

## Generated Content Features

### Panel Types
- **Timeseries** (40%) - Time-based metrics with realistic queries
- **Stat** (20%) - Single value indicators  
- **Gauge** (10%) - Visual progress indicators
- **Table** (10%) - Structured data display
- **Heatmap** (5%) - Distribution visualizations
- **Piechart** (5%) - Proportion displays
- **Bar Gauge** (5%) - Comparative metrics
- **Text** (5%) - Documentation panels

### Realistic Data Sources
- **Prometheus** - Metrics queries with rate(), histogram_quantile()
- **Loki** - Log queries with JSON parsing and filtering
- **Tempo** - Distributed tracing queries
- **Elasticsearch** - Full-text search and aggregations
- **MySQL/PostgreSQL** - Database metrics
- **InfluxDB** - Time-series data
- **CloudWatch** - AWS monitoring

### Template Variables
- **Environment** (prod, staging, dev)
- **Service** - Microservice selection
- **Instance** - Server/container filtering  
- **Region** - Geographic filtering
- **Cluster** - Kubernetes cluster selection
- **Job** - Prometheus job filtering
- **Namespace** - Kubernetes namespace filtering

### Dashboard Metadata
- **Realistic titles** - "Production Monitoring", "Critical Operations"
- **Appropriate tags** - monitoring, infrastructure, business, devops
- **Time ranges** - Last hour default with 30s refresh
- **Annotations** - Deployment events, incidents
- **Panel descriptions** - Helpful context for each visualization

## File Structure

```
perf/cmd/generate_dashboards/
├── main.go                 # Main generator logic
├── README.md              # This documentation
└── generated_dashboards/  # Output directory (created on run)
    ├── small-dashboard.json
    ├── medium-dashboard.json  
    ├── large-dashboard.json
    └── xlarge-dashboard.json
```

## Integration with Performance Testing

These dashboards are designed to work with the nanogit performance testing suite:

1. **Repository testing** - Use as varied file sizes in Git repositories
2. **JSON parsing** - Test JSON processing performance with complex structures
3. **Import/export** - Test Grafana dashboard import performance
4. **Memory usage** - Analyze memory consumption with large dashboard files
5. **Network transfer** - Test file transfer performance across different sizes

## Customization

To modify dashboard characteristics, edit the `GetDashboardSpecs()` function in `main.go`:

```go
{
    Name:           "custom",
    PanelCount:     100,        // Number of panels
    TemplateVars:   12,         // Number of variables
    SizeCategory:   "large",    // Size category
    TargetSizeKB:   500,        // Target file size
    HasAnnotations: true,       // Include annotations
    HasAlerts:      true,       // Include alert rules
    DataSources:    []string{"prometheus", "loki"}, // Available datasources
}
```

## Output Validation

The tool validates generated dashboards by:
- Ensuring valid JSON structure
- Including required Grafana schema fields
- Generating realistic panel configurations
- Creating proper datasource references
- Maintaining consistent ID assignments

Each generated file can be directly imported into Grafana for testing or development purposes.

## Dependencies

- Go 1.24+
- No external dependencies (uses only standard library)
- Compatible with all Grafana versions 8.0+