# Exoscale Terraform Provider

- Website: https://www.terraform.io
- [![Actions Status](https://github.com/exoscale/terraform-provider-exoscale/workflows/run-acceptance-tests/badge.svg?branch=master)](https://github.com/exoscale/terraform-provider-exoscale/actions?query=workflow%3Arun-acceptance-tests+branch%3Amaster)
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://raw.githubusercontent.com/hashicorp/terraform-website/master/public/img/logo-hashicorp.svg" width="600px">


## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.15+
- [Go](https://golang.org/doc/install) 1.16+ (to build the provider plugin)
- An [Exoscale](https://portal.exoscale.com/register) account


## Installation

### From the Terraform Registry (recommended)

The Exoscale provider is available on the [Terraform Registry][tf-exo-registry].
To use it, simply execute the `terraform init` command in a directory containing
Terraform configuration files referencing [Exoscale provider
resources][tf-exo-doc]:

```console
terraform init

Output:

Initializing the backend...

Initializing provider plugins...
- Finding exoscale/exoscale versions matching "0.18.2"...
- Installing exoscale/exoscale v0.18.2...
- Installed exoscale/exoscale v0.18.2 (signed by a HashiCorp partner, key ID 8B58C61D4FFE0C86)

...
```


### From Sources

If you prefer to build the plugin from sources, clone the GitHub repository
locally and run the command `make build` from the root of the sources directory.
Upon successful compilation, a `terraform-provider-exoscale_vdev` plugin binary
file can be found in the `bin/` directory. Then, follow the Terraform
documentation on [how to install provider plugins][tf-doc-provider-install].


## Usage

The complete and up-to-date documentation for the Exoscale provider is
available on the [Terraform Registry][tf-exo-doc].  Additionally, you can find
information on the general Terraform usage on the [HashiCorp Terraform
website][tf-doc].


## Contributing

* If you think you've found a bug in the code or you have a question regarding
  the usage of this software, please reach out to us by opening an issue in
  this GitHub repository.
* Contributions to this project are welcome: if you want to add a feature or a
  fix a bug, please do so by opening a Pull Request in this GitHub repository.
  In case of feature contribution, we kindly ask you to open an issue to
  discuss it beforehand.
* The documentation in the `docs` folder is generated from the descriptions
  in the source code. If you change the .Description of a resource attribute
  for example, you will need to run `go generate` in the root folder of the
  repository to update the generated docs. This is necessary to mere any PR
  as our CI checks whether the docs are up to date with the sources.
* Code changes require associated *acceptance tests*: the complete provider
  test suite (`make test-acc`) is executed as part of the project's GitHub
  repository [CI workflow][tf-exo-gh-ci], however you can execute targeted
  tests locally before submitting a Pull Request to ensure tests pass (e.g. for
  the `exoscale_compute` resource only):
* We are migrating the provider to the [new plugin framework](https://developer.hashicorp.com/terraform/plugin/framework). 
  If you'd like to implement new resources, please do so in the framework.
  The [zones datasource](./pkg/resources/zones/datasource.go) may provide the necessary inspiration.

```sh
make GO_TEST_EXTRA_ARGS="-v -run ^TestAccResourceCompute$" test-acc
```

### Development Setup

If you would like to use the terraform provider you have built and try
configurations on it as you are developing, then we recommend setting up
a `dev_override`. Create a file named `dev.tfrc` in the root directory
of this repository:

``` hcl
provider_installation {
  dev_overrides {
    "exoscale/exoscale" = "/path/to/the/repository/root/directory"
  }

  direct {}
}
```

Now `export TF_CLI_CONFIG_FILE=$PWD/dev.tfrc` in your shell and from now
on, whenever you run a `terraform` command in this shell and the configuration
references the `exoscale/exoscale` provider, it will use the provider you
built locally instead of downloading an official release. For this to work
you need to make sure you always run `go build` so that your changes are
compiled into a provider binary in the root directory of the repository.


[tf-doc-provider-install]: https://www.terraform.io/docs/configuration/provider-requirements.html#provider-installation
[tf-doc]: https://www.terraform.io/docs/index.html
[tf-exo-doc]: https://registry.terraform.io/providers/exoscale/exoscale/latest/docs
[tf-exo-gh-ci]: https://github.com/exoscale/terraform-provider-exoscale/actions?query=workflow%3ACI
[tf-exo-registry]: https://registry.terraform.io/providers/exoscale/exoscale/latest
