GO_FILES := $(shell find -type f -name "*.go" ! -name "*_templ.go")
TEMPL_FILES := $(shell find -type f -name "*.templ")
TEMPL_GO_FILES := $(TEMPL_FILES:.templ=_templ.go)
STATIC_DIR := internal/controller/web/static/ext
STATIC_FILES := $(patsubst %, $(STATIC_DIR)/%, htmx.js tailwind.js daisyui.css)

SOURCES := $(GO_FILES) $(TEMPL_FILES) $(STATIC_FILES)

wheelhouse: $(GO_FILES) $(TEMPL_GO_FILES) $(STATIC_FILES)
	go build ./cmd/wheelhouse

.PHONY: test
test: $(TEMPL_GO_FILES)
	go test ./...

.PHONY: coverage
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

$(STATIC_FILES):
	mkdir -p $(STATIC_DIR)
	wget -O $(STATIC_DIR)/htmx.js https://unpkg.com/htmx.org@2.0.4
	wget -O $(STATIC_DIR)/daisyui.css https://cdn.jsdelivr.net/npm/daisyui@5.0.0-beta.8/daisyui.css
	wget -O $(STATIC_DIR)/tailwind.js https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4
