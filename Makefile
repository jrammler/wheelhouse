GO_FILES := $(shell find -type f -name "*.go" ! -name "*_templ.go")
TEMPL_FILES := $(shell find -type f -name "*.templ")
TEMPL_GO_FILES := $(TEMPL_FILES:.templ=_templ.go)

SOURCES := $(GO_FILES) $(TEMPL_FILES)

wheelhouse: $(GO_FILES) $(TEMPL_GO_FILES)
	go build ./cmd/wheelhouse

.PHONY: test
test:
	go test ./...

.PHONY: coverag
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

.PHONY: run
run: wheelhouse
	./wheelhouse serve :8080 config.json

.PHONY: format
format:
	templ fmt .
	go fmt ./...

.PHONY: watch
watch: wheelhouse
	echo $(SOURCES) | tr " " "\n" | entr -dr make run

%_templ.go: %.templ
	templ generate -f $^
