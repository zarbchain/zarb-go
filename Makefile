PACKAGES=$(shell go list ./... | grep -v 'tests')
HERUMI= $(shell pwd)/.herumi
CGO_LDFLAGS=CGO_LDFLAGS="-L$(HERUMI)/bls/lib -lbls384_256 -lm -lstdc++ -g -O2"
BUILD_LDFLAGS= -ldflags "-X github.com/zarbchain/zarb-go/version.build=`git rev-parse --short=8 HEAD`"
RELEASE_LDFLAGS= -ldflags "-s -w"

all: install test

########################################
### Tools needed for development
devtools:
	@echo "Installing devtools"
	go install zombiezen.com/go/capnproto2/capnpc-go@v2.18.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.38.0
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.3.0
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.3.0
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.25.0
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0
	go install github.com/bufbuild/buf/cmd/buf@v0.39.1
	go install github.com/rakyll/statik@v0.1.7


herumi:
	@if [ ! -d $(HERUMI) ]; then \
		git clone --recursive git://github.com/herumi/bls.git $(HERUMI)/bls && cd $(HERUMI)/bls && make minimized_static; \
	fi


########################################
### Building
build: herumi
	$(CGO_LDFLAGS) go build $(BUILD_LDFLAGS) ./cmd/zarb

install: herumi
	$(CGO_LDFLAGS) go install $(BUILD_LDFLAGS) ./cmd/zarb

release: herumi
	$(CGO_LDFLAGS) go build $(RELEASE_LDFLAGS) ./cmd/zarb

########################################
### Testing
unit_test: herumi
	$(CGO_LDFLAGS) go test $(PACKAGES)

test: herumi
	$(CGO_LDFLAGS) go test ./... -covermode=atomic

test_race: herumi
	$(CGO_LDFLAGS) go test ./... --race

########################################
### Docker
docker:
	docker build --tag zarb -f ./containers/Dockerfile .

########################################
### capnp and proto
capnp:
	capnp compile \
		-ogo ./www/capnp/zarb.capnp

proto:
	cd www/grpc/ && buf generate --path ./proto/zarb.proto --path proto/payloads.proto
	# Generate static assets for OpenAPI UI
	cd www/grpc/ && statik -m -f -src third_party/OpenAPI/

########################################
### Formatting, linting, and vetting
fmt: herumi
	$(CGO_LDFLAGS) gofmt -s -w .
	$(CGO_LDFLAGS) golangci-lint run -e "SA1019" \
		--timeout=5m0s \
		--enable=gofmt \
		--enable=unconvert \
		--enable=unparam \
		--enable=golint \
		--enable=asciicheck \
		--enable=misspell \
		--enable=gosec

# To avoid unintended conflicts with file names, always add to .PHONY
# unless there is a reason not to.
# https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html
.PHONY: build install release
.PHONY: test unit_test test_race
.PHONY: devtools herumi capnp proto
.PHONY: fmt docker