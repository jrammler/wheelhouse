GO_FILES := $(shell find -type f -name "*.go" ! -name "*_templ.go")
TEMPL_FILES := $(shell find -type f -name "*.templ")
TEMPL_GO_FILES := $(TEMPL_FILES:.templ=_templ.go)
STATIC_DIR := internal/controller/web/static
STATIC_FILES := $(patsubst %, $(STATIC_DIR)/%, js/htmx.js css/tailwind.css css/daisyui.css)

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
	./wheelhouse serve :8080  ~/.config/wheelhouse/config.json

.PHONY: format
format:
	templ fmt .
	go fmt ./...

.PHONY: watch
watch: wheelhouse
	echo $(SOURCES) | tr " " "\n" | entr -dr make run

%_templ.go: %.templ
	templ generate -f $^

$(STATIC_DIR)/js/htmx.js:
	mkdir -p $(STATIC_DIR)/js
	curl -sL -o $(STATIC_DIR)/js/htmx.js https://unpkg.com/htmx.org@2.0.4

$(STATIC_DIR)/css/daisyui.css:
	mkdir -p $(STATIC_DIR)/css
	curl -sL -o $(STATIC_DIR)/css/daisyui.css https://cdn.jsdelivr.net/npm/daisyui@5.0.0-beta.8/daisyui.css

$(STATIC_DIR)/css/tailwind.css: internal/controller/web/templates/tailwind.css $(TEMPL_FILES)
	mkdir -p $(STATIC_DIR)/css
	./tailwindcss -i internal/controller/web/templates/tailwind.css -o $(STATIC_DIR)/css/tailwind.css --minify
	touch $(STATIC_DIR)/css/tailwind.css
