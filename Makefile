APP := kubecat
BUILD_DIR := build
SOURCES := $(shell find . -name '*.go')

.PHONY: all clean

all: $(BUILD_DIR)/$(APP)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/$(APP): $(SOURCES) | $(BUILD_DIR)
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP)

clean:
	rm -rf $(BUILD_DIR)
