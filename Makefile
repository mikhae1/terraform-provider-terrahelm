PROVIDER = terraform-provider-terrahelm
VERSION = 1.0.0

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

release:
	tfplugindocs

build:
	go build -o $(PROVIDER)

install: build
	@echo Installing $(PROVIDER) v$(VERSION)
	mkdir -p $(TF_HOME_PATH)
	cp $(PROVIDER) $(TF_HOME_PATH)/
