PROVIDER = terraform-provider-terrahelm

TF_ARCH := $(shell go env GOOS)_$(shell go env GOARCH)
TF_LOC_DIR = examples/terraform.d/plugins/local/mikhae1/terrahelm/0.1.0
TF_PATH = $(TF_LOC_DIR)/$(TF_ARCH)
export TF_LOG = INFO

all: tf-init tf-install

tf-init: build
	@cd examples &&\
		rm -rf .terraform.lock* .terraform &&\
		terraform init --upgrade

tf-plan:
	cd examples && terraform plan $(filter-out $@,$(MAKECMDGOALS))

tf-apply:
	cd examples && terraform apply --auto-approve

tf-clean:
	helm -n nginx uninstall nginx || true
	rm -rf examples/terraform.tfstate*

tf-install: build
	mkdir -p $(TF_PATH)
	ln -sf $(abspath $(PROVIDER)) $(TF_PATH)/$(PROVIDER)
	$(MAKE) tf-plan

build:
	go build -o $(PROVIDER)
