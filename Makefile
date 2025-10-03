NAME	:= isomer
VERSION := $(shell git describe --always --dirty="-dev")
ARCH    ?= amd64

export GOFLAGS += -mod=vendor -ldflags=-X=main.Version=$(VERSION)
export CLANG = clang
export STRIP = llvm-strip
export MAKEDIR  = $(CURDIR)

.SUFFIXES:
MAKEFLAGS+=-r

generated := internal/dispatcher_bpfel.go internal/dispatcher_bpfeb.go
deps := $(addsuffix .d,$(generated))

.PHONY: all
all: $(generated) $(deps)
	@mkdir -p "bin/$(ARCH)"
	GOARCH="$(ARCH)" go build -v -o "bin/$(ARCH)" ./cmd/...

internal/%_bpfel.go internal/%_bpfeb.go internal/%.go.d:
	go generate ./internal

.PHONY: package
package: $(NAME)_$(VERSION)_$(ARCH).deb

$(NAME)_$(VERSION)_%.deb: clean all
	TARGET_ARCH=$* VERSION="$(VERSION)" nfpm package -p deb -f nfpm.yaml

.PHONY: test
test: RUNNER=go test -exec sudo
test:
	$(RUNNER) -coverpkg=./... -coverprofile=coverage.out -count 1 $(TESTFLAGS) ./...

.PHONY: cover
cover:
	go tool cover -html coverage.out -o coverage.html

.PHONY: build-tests
build-tests:
	go list ./... | while read pkg; do go test -c $${pkg} || exit; done

.PHONY: lint
lint:
	test -z $$(gofmt -l $$(find . -name '*.go' ! -path './vendor/*'))

.PHONY: clean
clean:
	$(RM) -r bin deb $(deps)

.PHONY: distclean
distclean: clean
	$(RM) *.deb

ifneq ($(MAKECMDGOALS),clean)
-include $(deps)
endif
