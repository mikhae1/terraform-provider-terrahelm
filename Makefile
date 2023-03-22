tf_path = examples/terraform.d/plugins/local/mikhae1/terrahelm/0.1.0/darwin_arm64

all: build tf-reset tf-init tf-plan

tf-init:
	cd examples && terraform init --upgrade

tf-plan:
	cd examples && terraform plan

tf-reset:
	cd examples && rm -rf .terraform.lock* .terraform

build:
	mkdir -p $(tf_path)
	go build -o $(tf_path)
