# Exoscale Terraform Provider

- Website: https://www.terraform.io
- [![Actions Status](https://github.com/exoscale/terraform-provider-exoscale/workflows/CI/badge.svg)](https://github.com/exoscale/terraform-provider-exoscale/actions?query=workflow%3ACI)
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">


## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.12+
- [Go](https://golang.org/doc/install) 1.13+ (to build the provider plugin)
- An [Exoscale](https://portal.exoscale.com/register) account


## Installation

### From the Terraform Registry (recommended)

The Exoscale provider is available on the [Terraform Registry][tf-exo-registry].
To use it, simply execute the `terraform init` command in a directory containing
Terraform configuration files referencing [Exoscale provider
resources][tf-exo-doc]:

```console
$ terraform init

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
* Code changes require associated *acceptance tests*: the complete provider
  test suite (`make test-acc`) is executed as part of the project's GitHub
  repository [CI workflow][tf-exo-gh-ci], however you can execute targeted
  tests locally before submitting a Pull Request to ensure tests pass (e.g. for
  the `exoscale_compute` resource only):

```sh
make GO_TEST_EXTRA_ARGS="-v -run ^TestAccResourceCompute$" test-acc
```


[tf-doc-provider-install]: https://www.terraform.io/docs/configuration/provider-requirements.html#provider-installation
[tf-doc]: https://www.terraform.io/docs/index.html
[tf-exo-doc]: https://registry.terraform.io/providers/exoscale/exoscale/latest/docs
[tf-exo-gh-ci]: https://github.com/exoscale/terraform-provider-exoscale/actions?query=workflow%3ACI
[tf-exo-registry]: https://registry.terraform.io/providers/exoscale/exoscale/latest
