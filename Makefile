SCRIPTS_DIR := $(HOME)/Development/github.com/rios0rios0/pipelines
REPO_URL    := https://github.com/rios0rios0/pipelines.git

.PHONY: all scripts lint lint-fix horusec test build install uninstall docs
all: lint horusec semgrep gitleaks test

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

horusec: scripts
	$(SCRIPTS_DIR)/global/scripts/horusec/run.sh .

semgrep: scripts
	$(SCRIPTS_DIR)/global/scripts/semgrep/run.sh "golang"

gitleaks: scripts
	$(SCRIPTS_DIR)/global/scripts/gitleaks/run.sh .

test: scripts
	$(SCRIPTS_DIR)/global/scripts/golang/test/run.sh .


VERSION = 2.2.0

docs:
	export GOBIN=$PWD/bin
	export PATH=$GOBIN:$PATH
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	tfplugindocs generate

build:
	go build -o bin/terraform-provider-http

install:
	make build
	#		 ~/.terraform.d/plugins/${host_name}/${namespace}/${type}/${version}/${target}
	mkdir -p ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/
	cp bin/terraform-provider-http ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/

uninstall:
	rm -rf ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/
