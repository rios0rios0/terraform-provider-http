build:
	go build -o bin/terraform-provider-http

install:
	make build
	mkdir -p ~/.terraform.d/plugins/local/http/1.0.0/linux_amd64
	cp bin/terraform-provider-http ~/.terraform.d/plugins/local/http/1.0.0/linux_amd64/

docs:
	export GOBIN=$PWD/bin
	export PATH=$GOBIN:$PATH
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	tfplugindocs generate
