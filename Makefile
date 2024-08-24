build:
	go build -o bin/terraform-provider-http

install:
	make build
	#		 ~/.terraform.d/plugins/${host_name}/${namespace}/${type}/${version}/${target}
	mkdir -p ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/0.0.7/linux_amd64/
	cp bin/terraform-provider-http ~/.terraform.d/plugins/hashicorp-local.com/rios0rios0/http/0.0.7/linux_amd64/

docs:
	export GOBIN=$PWD/bin
	export PATH=$GOBIN:$PATH
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	tfplugindocs generate
