BIN_PATH := bin
TARGET_NAME := sbbeautify
TARGET := $(BIN_PATH)/$(TARGET_NAME)

${TARGET}: *.go
	go build -o $@

.PHONY: build
build: $(TARGET)

.PHONY: run
run:
	go run .
