DEFAULT_TARGET: build
CURRENT_DIR=$(shell pwd)
BUILD_DIR=build
BINARY_NAME=dcos-check-runner
PKG_NAME=$(BINARY_NAME)
IMAGE_NAME=dcos-check-runner-dev

all: test install

.PHONY: docker
docker:
	docker build -t $(IMAGE_NAME) .

.PHONY: build
build: docker
	mkdir -p $(BUILD_DIR)
	docker run \
		-v $(CURRENT_DIR):/$(BINARY_NAME) \
		-w /$(BINARY_NAME) \
		--privileged \
		--rm \
		$(IMAGE_NAME) \
		go build -mod=vendor -v -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: test
test: docker
	docker run \
		-v $(CURRENT_DIR):/$(BINARY_NAME) \
		-w /$(BINARY_NAME) \
		--privileged \
		--rm \
		$(IMAGE_NAME) \
		bash -x -c './scripts/test.sh'

.PHONY: shell
shell:
	docker run \
		-v $(CURRENT_DIR):/$(BINARY_NAME) \
		-w /$(BINARY_NAME) \
		--privileged \
		--rm \
		-it \
		$(IMAGE_NAME) \
		/bin/bash

# install does not run in a docker container because it only compiles on linux.
.PHONY: install
install:
	go install -v -ldflags '$(LDFLAGS)'

.PHONY: clean
clean:
	rm -rf ./$(BUILD_DIR)
