module github.com/grafana/nanogit/testutil/examples

go 1.24

require (
	github.com/grafana/nanogit v0.3.9
	github.com/grafana/nanogit/testutil v0.1.0
	github.com/onsi/ginkgo/v2 v2.22.2
	github.com/onsi/gomega v1.37.0
	github.com/stretchr/testify v1.11.1
)

// During development, use replace directives
replace github.com/grafana/nanogit => ../../

replace github.com/grafana/nanogit/testutil => ../
