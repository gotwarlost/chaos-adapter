
.DEFAULT_GOAL := build

### ====== begin: Deployment parameters =====
export NAMESPACE ?= chaos
export SYSTEM_NAMESPACE ?= istio-system
export IMAGE ?= gotwarlost/chaos-adapter:latest
export REPLICAS ?= 3
export SERVER_REPLICAS ?= 2
export CLIENT_REPLICAS ?= 5
export PROXY_INBOUND_PORTS ?= 4080,8080
export HANDLER_ENDPOINT ?= chaos-adapter.$(NAMESPACE):4080
### ====== end: Deployment parameters =====

.PHONY: deploy
deploy:
	@envsubst < k8s/definitions.yaml
	@envsubst < k8s/adapter.yaml
	@envsubst < k8s/server.yaml
	@envsubst < k8s/client.yaml

### the rest of the crap builds the code and docker image and doesn't need looking into if you are using the
### image already pushed

ISTIO_VERSION=1.1.15
ISTIO_REPO=git@github.com:istio/api.git

DIR := ${CURDIR}

# special case for protoc because they use osx instead of darwin
PLATFORM_OSX := $(shell uname | sed 's/Darwin/osx/' | tr '[:upper:]' '[:lower:]')
OS_CMD = $(shell [[ "$$(uname)" = "Darwin" ]]  && echo b || echo w)

CHAOS_DIR=adapter/chaos
CONFIG_DIR=adapter/config

TEMPLATE_DS_SET=$(CHAOS_DIR)/template.descriptor_set
HANDLER_DS_SET=$(CHAOS_DIR)/template_handler_service.descriptor_set
CONFIG_DS_SET=$(CONFIG_DIR)/config.descriptor_set

GOGO_PROTOBUF_VERSION=v1.3.0
GOGO_PROTOBUF_REPO=git@github.com:gogo/protobuf.git
GOGO_GOOGLE_PROTOBUF_REPO=git@github.com:gogo/googleapis.git

PROTO_LIB=proto-vendor

.PHONY: install-protoc
install-protoc: VERSION = 3.7.1
install-protoc:
	@echo +install protoc
	curl -L https://github.com/protocolbuffers/protobuf/releases/download/v${VERSION}/protoc-${VERSION}-${PLATFORM_OSX}-x86_64.zip -o protoc.zip
	mkdir -p protoc
	unzip -o protoc.zip -d ./protoc
	rm protoc.zip
	@echo +install gogoslick
	go install github.com/gogo/protobuf/protoc-gen-gogoslick

.PHONY: codegen-libs
codegen-libs:
	@echo +fetch dependent protobuf libraries
	@rm -rf $(PROTO_LIB)
	@mkdir -p $(PROTO_LIB)
	@git clone --depth 1 -b $(ISTIO_VERSION) $(ISTIO_REPO) $(PROTO_LIB)/istio.io/api
	@git clone --depth 1 -b $(GOGO_PROTOBUF_VERSION) $(GOGO_PROTOBUF_REPO) $(PROTO_LIB)/gogo
	@git clone --depth 1 -b $(GOGO_PROTOBUF_VERSION) $(GOGO_GOOGLE_PROTOBUF_REPO) $(PROTO_LIB)/googleapis

.PHONY: codegen
codegen: PROTO_ARGS=--proto_path=$(PROTO_LIB)/istio.io/api --proto_path=$(PROTO_LIB)/gogo --proto_path=$(PROTO_LIB)/googleapis --include_imports --include_source_info
codegen: GOGO_ARGS=--gogoslick_out=plugins=grpc,Mgogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto,Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/rpc/status.proto=github.com/gogo/googleapis/google/rpc,Mgoogle/rpc/code.proto=github.com/gogo/googleapis/google/rpc,Mgoogle/rpc/error_details.proto=github.com/gogo/googleapis/google/rpc
codegen: MIXER_ARGS=-m google/protobuf/any.proto:github.com/gogo/protobuf/types -m gogoproto/gogo.proto:github.com/gogo/protobuf/gogoproto -m google/protobuf/duration.proto:github.com/gogo/protobuf/types -m google/protobuf/timestamp.proto:github.com/gogo/protobuf/types -m google/rpc/status.proto:github.com/gogo/googleapis/google/rpc -m google/protobuf/struct.proto:github.com/gogo/protobuf/types

codegen: codegen-libs
	@echo +compile template protobuf
	@./protoc/bin/protoc $(PROTO_ARGS) $(GOGO_ARGS):$(CHAOS_DIR) --proto_path=$(CHAOS_DIR)  --descriptor_set_out=$(TEMPLATE_DS_SET) $(CHAOS_DIR)/template.proto
	@./protoc/bin/protoc $(PROTO_ARGS) $(GOGO_ARGS):$(CONFIG_DIR) --proto_path=$(CONFIG_DIR)  --descriptor_set_out=$(CONFIG_DS_SET) $(CONFIG_DIR)/config.proto
	@echo +create mixer code
	@docker run -v $(DIR):/build --workdir /build istio/mixer_codegen:$(ISTIO_VERSION) api $(MIXER_ARGS) \
		-t $(TEMPLATE_DS_SET) \
		--go_out $(CHAOS_DIR)/template_handler_service.gen.go \
		--proto_out $(CHAOS_DIR)/template_handler_service.proto
	@echo +compile handler protobuf
	@./protoc/bin/protoc $(PROTO_ARGS) $(GOGO_ARGS):$(CHAOS_DIR) --proto_path=$(CHAOS_DIR) --descriptor_set_out=$(HANDLER_DS_SET) $(CHAOS_DIR)/template_handler_service.proto
	@echo +create base64 template
	@cat $(CONFIG_DS_SET) | base64 -$(OS_CMD)0 > $(CONFIG_DS_SET).b64
	@cat $(HANDLER_DS_SET) | base64 -$(OS_CMD)0 > $(HANDLER_DS_SET).b64

build:
	go install ./...

clean:
	find $(CONFIG_DIR) -type f -not -name config.proto | xargs rm
	find $(CHAOS_DIR) -type f -not -name template.proto | xargs rm
	rm -rf $(PROTO_LIB)
	rm -rf vendor/


export ADAPTER_BASE64_CONFIG := $(shell cat $(CONFIG_DS_SET).b64)
export TEMPLATE_BASE64_CONFIG := $(shell cat $(HANDLER_DS_SET).b64)

.PHONY: docker
docker:
	go mod vendor
	docker build -t $(IMAGE) .
	docker push $(IMAGE)


