module github.com/teracrafts/huefy-sdk-go/v2

go 1.21

require (
	github.com/stretchr/testify v1.8.4
	github.com/teracrafts/huefy-sdk/core/kernel v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/teracrafts/huefy-sdk/core/kernel => ../../core/kernel
