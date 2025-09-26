VERSION = 2.2.0
SCRIPTS_DIR := $(HOME)/Development/github.com/rios0rios0/pipelines
REPO_URL    := https://github.com/rios0rios0/pipelines.git

.PHONY: build install uninstall docs test scripts lint lint-fix
default: test

build:
	go build -o bin/terraform-provider-http

install:
	make build
	#		 ~/.terraform.d/plugins/${host_name}/${namespace}/${type}/${version}/${target}
	mkdir -p ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/
	cp bin/terraform-provider-http ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/

uninstall:
	rm -rf ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/

scripts:
	if [ ! -d "$(SCRIPTS_DIR)" ]; then \
	  git clone $(REPO_URL) $(SCRIPTS_DIR); \
	else \
	  cd $(SCRIPTS_DIR) && git pull; \
	fi

lint: scripts
	$(SCRIPTS_DIR)/global/scripts/golangci-lint/run.sh .

lint-fix: scripts
	$(SCRIPTS_DIR)/global/scripts/golangci-lint/run.sh --fix .

docs:
	export GOBIN=$PWD/bin
	export PATH=$GOBIN:$PATH
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	tfplugindocs generate

test:
	TF_ACC=1 go test -v -tags test,unit,integration -coverpkg=./... -covermode=count -coverprofile=coverage.txt -timeout 120m ./... | grep -v '\[no test files\]'
	go tool cover -func coverage.txt
