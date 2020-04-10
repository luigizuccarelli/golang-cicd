.PHONY: all test build

all: clean test build

build: 
	mkdir -p build
	go build -o build ./... 

test:
	mv cmd .cmd
	go test -v -tags=test  -coverprofile=tests/results/cover.out ./...
	mv .cmd cmd

cover:
	go tool cover -html=tests/results/cover.out -o tests/results/cover.html

clean:
	rm -rf build/*
	go clean ./...
