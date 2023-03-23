PROVIDER_NAME = terraform-provider-terrahelm

TF_ARCH := $(shell go env GOOS)_$(shell go env GOARCH)
TF_LOC_DIR = examples/terraform.d/plugins/local/mikhae1/terrahelm/0.1.0
TF_PATH = $(TF_LOC_DIR)/$(TF_ARCH)


all: build tf-init tf-plan

tf-init:
	cd examples && terraform init --upgrade

tf-plan:
	cd examples && terraform plan

tf-clean:
	cd examples && rm -rf .terraform.lock* .terraform

build:
	go build -o $(PROVIDER_NAME)

install: build
	mkdir -p $(TF_PATH)
	ln -sf $(abspath $(PROVIDER_NAME)) $(TF_PATH)/$(PROVIDER_NAME)
	$(MAKE) tf-plan
