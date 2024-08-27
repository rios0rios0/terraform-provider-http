.PHONY: build install uninstall docs test
default: test

build:
	go build -o bin/terraform-provider-http

install:
	make build
	#		 ~/.terraform.d/plugins/${host_name}/${namespace}/${type}/${version}/${target}
	mkdir -p ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/1.0.0/linux_amd64/
	cp bin/terraform-provider-http ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/1.0.0/linux_amd64/

uninstall:
	rm -rf ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/1.0.0/linux_amd64/

docs:
	export GOBIN=$PWD/bin
	export PATH=$GOBIN:$PATH
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	tfplugindocs generate

test:
	TF_ACC=1 go test -v -tags test,unit,integration -coverpkg=./... -covermode=count -coverprofile=coverage.txt -timeout 120m ./... | grep -v '\[no test files\]'
	go tool cover -func coverage.txt
