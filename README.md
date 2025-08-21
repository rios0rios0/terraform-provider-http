# Terraform Provider for HTTP Requests
This Terraform provider facilitates the execution of HTTP requests and enables the storage of responses within the Terraform state.
The primary advantage of this feature is the ability to utilize the stored responses in subsequent requests.

While this provider is not designed to replace the [http](https://registry.terraform.io/providers/hashicorp/http/latest/docs) provider, it can be used alongside it.
Notably, the [http](https://registry.terraform.io/providers/hashicorp/http/latest/docs) provider does not store responses in the state, which limits its ability to use responses in future requests.

This provider supports specifying the URL, method, and headers, and it captures both the response body and response code.

## Requirements
- [Go](https://golang.org/doc/install) >= 1.23.4
- [Terraform](https://www.terraform.io/downloads.html) >= 1.10.4 (tested and approved)

## Contributing
Contributions are welcome! Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information.

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Development Status

For planned features, improvements, and known issues, please check our [GitHub Issues](https://github.com/rios0rios0/terraform-provider-http/issues).

Current focus areas include:
- Resource lifecycle management (delete functionality)
- Enhanced validation and state management
- Documentation improvements
- Performance optimizations

## References
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework/resources/create)
- [Develop a Terraform provider (Terraform HashiCups Provider)](https://github.com/hashicorp/terraform-provider-hashicups)
- [Terraform Provider Scaffolding (Terraform Plugin Framework)](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout/tree/master?tab=readme-ov-file)
