module github.com/kaltenecker-kg/hrobot-go/v2

// Keep the go directive without a patch component. actions/setup-go reads it
// from go.mod and treats "1.26" as a range (latest 1.26.x patch, with
// check-latest) but "1.26.0" as an exact version, which would pin CI to the
// first 1.26 release and skip standard-library security fixes. Indirect,
// test-only dependencies whose own go.mod declares 1.26.0 (golang.org/x/text
// >= v0.42.0, go-openapi/jsonpointer >= v1.0.0) would force the patch form
// through `go mod tidy`; they are held back for that reason. It is also why
// govulncheck is run with `go run ...@latest` rather than as a `tool`
// dependency: golang.org/x/vuln declares 1.26.0 too.
go 1.26

require github.com/getkin/kin-openapi v0.149.0

require (
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	golang.org/x/text v0.41.0 // indirect
)
