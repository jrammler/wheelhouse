GO_FILES := $(shell find -type f -name "*.go" ! -name "*_templ.go")
TEMPL_FILES := $(shell find -type f -name "*.templ")
TEMPL_GO_FILES := $(TEMPL_FILES:.templ=_templ.go)

SOURCES := $(GO_FILES) $(TEMPL_FILES)

main: $(GO_FILES) $(TEMPL_GO_FILES)
	go build cmd/wheelhouse/main.go

.PHONY: test
test:
	go test ./...

.PHONY: coverag
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

.PHONY: run
run: main
	./main 8080

.PHONY: format
format:
	templ fmt .
	go fmt ./...

.PHONY: watch
watch: main
	echo $(SOURCES) | tr " " "\n" | entr -dr make run

%_templ.go: %.templ
	templ generate -f $^
