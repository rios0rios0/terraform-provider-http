SCRIPTS_DIR := $(HOME)/Development/github.com/rios0rios0/pipelines
REPO_URL    := https://github.com/rios0rios0/pipelines.git

.PHONY: all scripts lint lint-fix semgrep gitleaks test build install uninstall docs
all: lint semgrep gitleaks test

scripts:
	if [ ! -d "$(SCRIPTS_DIR)" ]; then \
	  git clone $(REPO_URL) $(SCRIPTS_DIR); \
	else \
	  cd $(SCRIPTS_DIR) && git pull; \
	fi

lint: scripts
	$(SCRIPTS_DIR)/global/scripts/languages/golang/golangci-lint/run.sh .

lint-fix: scripts
	$(SCRIPTS_DIR)/global/scripts/languages/golang/golangci-lint/run.sh --fix .

semgrep: scripts
	$(SCRIPTS_DIR)/global/scripts/tools/semgrep/run.sh "golang"

gitleaks: scripts
	$(SCRIPTS_DIR)/global/scripts/tools/gitleaks/run.sh .

test: scripts
	$(SCRIPTS_DIR)/global/scripts/languages/golang/test/run.sh .


VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "dev")
LDFLAGS := -X main.version=$(VERSION)

docs:
	export GOBIN=$PWD/bin
	export PATH=$GOBIN:$PATH
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	tfplugindocs generate

build:
	go build -ldflags "$(LDFLAGS) -s -w" -o bin/terraform-provider-http

install:
	make build
	#		 ~/.terraform.d/plugins/${host_name}/${namespace}/${type}/${version}/${target}
	mkdir -p ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/
	cp bin/terraform-provider-http ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/

uninstall:
	rm -rf ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/$(VERSION)/linux_amd64/
