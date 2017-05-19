NAME=keto-tokens
AUTHOR=ukhomeofficedigital
REGISTRY=quay.io
GOVERSION ?= 1.8.1
ROOT_DIR=${PWD}
HARDWARE=$(shell uname -m)
GIT_SHA=$(shell git --no-pager describe --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
VERSION ?= $(shell awk '/version.*=/ { print $$3 }' doc.go | sed 's/"//g')
DEPS=$(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
PACKAGES=$(shell go list ./... | grep -v vendor)
LFLAGS ?= -X main.gitsha=${GIT_SHA}

.PHONY: test build docker static release lint cover vet all

default: build

golang:
	@echo "--> Go Version"
	@go version

build:
	@echo "--> Compiling the project"
	@mkdir -p bin
	go build -ldflags "${LFLAGS}" -o bin/${NAME}

static: golang deps
	@echo "--> Compiling the static binary"
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags "-w ${LFLAGS}" -o bin/${NAME}

docker:
	@echo "--> Building the container"
	docker build -t ${REGISTRY}/${AUTHOR}/${NAME}:${VERSION} .

release: static
	mkdir -p release
	gzip -c bin/${NAME} > release/${NAME}_${VERSION}_linux_${HARDWARE}.gz
	rm -f release/${NAME}

clean:
	rm -rf ./bin 2>/dev/null
	rm -rf ./release 2>/dev/null

deps:
	@echo "--> Installing build dependencies"
	go get github.com/Masterminds/glide

vet:
	@echo "--> Running go vet $(VETARGS) ."
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@go vet $(PACKAGES)

gofmt:
	@echo "--> Running gofmt check"
	@gofmt -s -l *.go \
	    | grep -q \.go ; if [ $$? -eq 0 ]; then \
            echo "You need to runn the make format, we have file unformatted"; \
            gofmt -s -l *.go; \
            exit 1; \
	    fi

format:
	@echo "--> Running go fmt"
	gofmt -s -w *.go

bench:
	@echo "--> Running go bench"
	godep go test -v -bench=.

coverage:
	@echo "--> Running go coverage"
	go test $(PACKAGES) -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

cover:
	@echo "--> Running go cover"
	@go test $(PACKAGES) --cover

test: deps
	@echo "--> Running the tests"
	@go test -v $(PACKAGES)
	@$(MAKE) gofmt
	@$(MAKE) vet
	@$(MAKE) cover

all: deps
	echo "--> Running all tests"
	glide install --strip-vendor
	@$(MAKE) test
	@$(MAKE) build
