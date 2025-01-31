GO_FILES := $(shell find -type f -name "*.go" ! -name "*_templ.go")
TEMPL_FILES := $(shell find -type f -name "*.templ")
TEMPL_GO_FILES := $(TEMPL_FILES:.templ=_templ.go)

SOURCES := $(GO_FILES) $(TEMPL_FILES)

main: $(GO_FILES) $(TEMPL_GO_FILES)
	go build cmd/wheelhouse/main.go

.PHONY: run
run: main
	./main 8080

.PHONY: format
format:
	templ fmt .
	echo $(GO_FILES) | xargs -n 1 go fmt

.PHONY: watch
watch: main
	echo $(SOURCES) | tr " " "\n" | entr -dr make run

%_templ.go: %.templ
	templ generate -f $^
