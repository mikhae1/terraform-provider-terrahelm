PROVIDER = terraform-provider-terrahelm
VERSION = 1.0.0
RELEASE_DIR = release
RELEASE_PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

GOX_BIN :=$(shell go env GOPATH)/bin/gox

TF_DEBUG_DIR = examples/local
TF_ARCH := $(shell go env GOOS)_$(shell go env GOARCH)
TF_LOCAL_PATH := $(TF_DEBUG_DIR)/terraform.d/plugins/local/mikhae1/terrahelm/$(VERSION)/$(TF_ARCH)
TF_HOME_PATH := $(HOME)/.terraform.d/plugins/github.com/mikhae1/terrahelm/$(VERSION)/$(TF_ARCH)
TF_ARGS = --auto-approve

export TF_LOG = INFO

all: tf-init tf-install

tf-init: build
	@cd $(TF_DEBUG_DIR) &&\
		rm -rf .terraform.lock* .terraform &&\
		terraform init --upgrade

tf-plan:
	cd $(TF_DEBUG_DIR) && terraform plan $(filter-out $@,$(MAKECMDGOALS))

tf-apply:
	cd $(TF_DEBUG_DIR) && terraform apply $(TF_ARGS)

tf-clean:
	cd $(TF_DEBUG_DIR) && terraform destroy $(TF_ARGS) || true
	rm -rf $(TF_DEBUG_DIR)/terraform.tfstate*

tf-install: build
	mkdir -p $(TF_LOCAL_PATH)
	ln -sf $(abspath $(PROVIDER)) $(TF_LOCAL_PATH)/$(PROVIDER)
	$(MAKE) tf-plan

test:
	go test -v ./provider

clean:
	rm -rf $(RELEASE_DIR)/* $(PROVIDER)

build-release: clean build test
	@echo "=> Building release binaries..."
	@mkdir -p $(RELEASE_DIR)
	$(GOX_BIN) -osarch="$(RELEASE_PLATFORMS)" -output="$(RELEASE_DIR)/{{.OS}}-{{.Arch}}/$(PROVIDER)"

release: build-release
	tfplugindocs || true
	@echo "=> Creating release packages..."
	@for platform in $(shell echo "$(RELEASE_PLATFORMS)" | tr ' ' '\n'); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		zip -r $(RELEASE_DIR)/$(PROVIDER)-$(VERSION)-$$os-$$arch.zip $(RELEASE_DIR)/$$os-$$arch; \
	done

build:
	go install
	go build -o $(PROVIDER)

install: build
	@echo Installing $(PROVIDER) v$(VERSION)
	mkdir -p $(TF_HOME_PATH)
	cp $(PROVIDER) $(TF_HOME_PATH)/

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi
