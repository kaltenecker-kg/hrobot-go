module github.com/kaltenecker-kg/hrobot-go/v2

// Keep the go directive without a patch component. actions/setup-go reads it
// from go.mod and treats "1.27" as a range (latest 1.27.x patch, with
// check-latest) but "1.27.0" as an exact version, which would pin CI to the
// first 1.27 release and skip standard-library security fixes. A dependency
// whose own go.mod declares a higher patch-form directive (e.g. "1.27.1")
// would force that form through `go mod tidy`; check the directive after
// bumping one.
go 1.27

require github.com/getkin/kin-openapi v0.153.0

require (
	github.com/go-openapi/jsonpointer v1.0.3 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	go.yaml.in/yaml/v4 v4.0.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)
