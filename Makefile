PKG=DMS-Prosecutor-go

# Used to populate variables in version package.
VERSION=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always)
REVISION=$(shell git rev-parse HEAD)$(shell if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)

GOCMD=go
GOCLEAN=$(GOCMD) clean
GO_LDFLAGS=-ldflags '-s -w -X $(PKG)/version.Version=$(VERSION) -X $(PKG)/version.Revision=$(REVISION)'


BINARY_NAME=prosecutor
SRC_FILE=./main.go

all: build

build:
	$(GOCMD) build $(GO_LDFLAGS) -o $(BINARY_NAME) $(SRC_FILE)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
