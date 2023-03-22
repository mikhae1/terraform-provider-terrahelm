all: build tf-reset tf-init tf-plan

tf-init:
	cd terraform && terraform init --upgrade

tf-plan:
	cd terraform && terraform plan

tf-reset:
	cd terraform && rm -rf .terraform.lock* .terraform

build:
	go build .
