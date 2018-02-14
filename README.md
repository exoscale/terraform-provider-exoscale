# Terraform provider for Exoscale

[![Build Status](https://travis-ci.org/exoscale/terraform-provider-exoscale.svg?branch=master)](https://travis-ci.org/exoscale/terraform-provider-exoscale)

## Installation

1. Download `terraform-provider-exoscale` from the [releases page](https://github.com/exoscale/terraform-provider-exoscale/releases);
2. Put it into the `.terraform/plugins/(darwin|linux|windows)_amd64` folder;
3. Run `terraform init`.

```
$ terraform providers
.
└── provider.exoscale
```

Go read the article on our weblog [Terraform on Exoscale](https://www.exoscale.com/syslog/2016/09/14/terraform-with-exoscale/).

## Usage

What follows below is the usage instructions for fully utilizing the Exoscale
resource plugin.  Additional documentation can be found in the examples directory.

### Provider requirements

```hcl
provider "exoscale" {
  version = "~> 0.9"
  token = "EXO..."
  secret = "..."

  timeout = 60  # default: waits 60 seconds in total for a resource
  delay = 5     # default: waits 5 seconds between each poll request
}

# or

provider "exoscale" {
  version = "~> 0.9"

  config = "cloudstack.ini"   # default: filename
  profile = "cloudstack"      # default: section name
}
```

You are required to provide at least the API token and secret key in order
to make use of the remaining Terraform resources.

The `timeout` is the maximum amount of time (in seconds, default: `60`) to wait
for async tasks to complete. Currently, this is used during the creation of
`compute` and `anti-affinity` resources.

### `cloudstack.ini`

```ini
[cloudstack]

endpoint = "https://api.exoscale.ch/compute"
key = "EXO..."
token = "..."
```

### Environment variables

You can specify the environment variables for these using
- **`token`**: ```EXOSCALE_KEY```, ```EXOSCALE_API_KEY```, ```CLOUDSTACK_KEY```, or ```CLOUDSTACK_API_KEY```;
- **`secret`**: ```EXOSCALE_SECRET```, ```EXOSCALE_SECRET_KEY```, ```CLOUDSTACK_SECRET```, or ```CLOUDSTACK_SECRET_KEY```;
- **`config`**: ```EXOSCALE_CONFIG```, or ```CLOUDSTACK_CONFIG```;
- **`profile`**: ```EXOSCALE_PROFILE```, or ```CLOUDSTACK_PROFILE```;
- **`timeout`** and **`delay`**: `EXOSCALE_TIMEOUT` and `EXOSCALE_DELAY` manages the asynchronous calls.
- **`compute_endpoint`**: ```EXOSCALE_COMPUTE_ENDPOINT```, or ```CLOUDSTACK_ENDPOINT```;
- **`dns_endpoint`**: ```EXOSCALE_DNS_ENDPOINT```.

## Resources

### Compute

```hcl
resource "exoscale_compute" "mymachine" {
  display_name = "mymachine"
  template = "Linux Debian 9 64-bit"
  size = "Medium"
  disk_size = 10
  key_pair = "me@mymachine"
  state = "Running"

  affinity_groups = []
  security_groups = ["default"]

  tags {
    production = "true"
  }
}
```

Attributes:

- **`display_name`**: initial `hostname`
- **`template`**: name from [the template](https://www.exoscale.com/templates/) 
- **`size`**: size of [the instances](https://www.exoscale.com/pricing/#/compute/), e.g. Tiny, Small, Medium, Large, etc.
- **`disk_size`**: size of the root disk in GiB (at least 10)
- **`zone`**: name of [the data-center](https://www.exoscale.com/datacenters/)
- `user_data`: [cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration
- **`key_pair`**: name of the SSH key pair to be installed
- `keyboard`: keyboard configuration (at creation time only)
- `state`: state of the virtual machine. E.g. `Running` or `Stopped`
- `affinity_groups`: list of [Affinity Groups](#affinity-groups)
- `security_groups`: list of [Security Groups](#security-groups)
- `tags`: dictionary of tags (key / value)

Values:

- `name`: name of the machine (`hostname`)
- `ip_address`: IP Address of the main network interface
- `virtual_machines_id`: list of the Compute instance members of the Affinity Group

### Security Group

```hcl
resource "exoscale_security_group" "http" {
  name = "HTTP"
  description = "Long text"

  tags {
    kind = "web"
  }
}

resource "exoscale_security_group_rule" "http" {
  security_group_id = "${exoscale_security_group.http.id}"
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"
  start_port = 80
  end_port = 80
}
```

Attributes:

- **`name`**: name of the security group 
- `description`: longer description
- `tags`: dictionary of tags (key / value)

Rule attributes:

- **`security_group_id`**: which security group the rule applies to
- **`protocol`**: the protocol, e.g. `TCP`, `UDP`, `ICMP`, etc.
- **`type`**: traffic type, either `INGRESS` or `EGRESS`
- `description`: human description
- `start_port`, `end_port`: for `TCP`, `UDP` traffic
- `icmp_type`, `icmp_code`: for `ICMP` traffic
- `cidr`: source/destination of the traffic as an IP subnet (conflicts with `user_security_group`)
- `user_security_group`: source/destination of the traffic as a security group (conflicts with `cidr`)

### (Anti-)Affinity Group

Define an affinity group. Anti-affinity groups make sure than the virtual machines are not running on the same physical host.

```hcl
resource "exoscale_affinity" "affinitylabel" {
  name = "affinity name"
  description = "long text"
  type = "host anti-affinity"
}
```

Attributes:

- **`name`**: name of the (anti-)affinity group 
- `description`: longer descriptions
- `type`: type of the anti-affinity groups

### SSH Resource

Declare an ssh key that will be used for any current/future instances

```hcl
resource "exoscale_ssh" "keylabel" {
  name = "keyname"
  key = "keycontents"
}
```

* ```name``` Defines the label in Exoscale to define the key
* ```key``` The ssh public key that will be copied into instances declared

### Elastic IP address

```
resource "exoscale_ipaddress" "myip" {
  ip_address = "159.100.251.224"
  zone = "ch-dk-2"
  tags {
    usage = "load-balancer"
  }
}
```

Attributes:

- **`zone`**: name of [the data-center](https://www.exoscale.com/datacenters/)
- `tags`: dictionary of tags (key / value)

Values:

- `ip_address`: IP address

**NB:** it's possible to `import` the IP address resource using the IP itself rather than the ID.

### Network

```hcl
resource "exoscale_network" "privNet" {
  name = "myPrivNet"
  display_text = "description"
  zone = "ch-dk-2"
  network_offering = "privNet"

  tags {
    # ...
  }
}

```

Attributes:

- **`name`** name of the network
- **`display_text`** description of the network
- `tags`: dictionary of tags (key / value)

## NIC

```hcl
resource "exoscale_nic" "eth1" {
  compute_id = "${exoscale_compute.mymachine.id}"
  network_id = "${exoscale_network.privNet.id}"
}
```

Attributes:

- **`compute_id`**: ID of the compute instance
- **`network_id`**: ID of the network instance

Values:

- `mac_address`: physical address of the network interface

### DNS

```hcl
resource "exoscale_domain" "exo" {
  name = "exo.exo"
}

resource "exoscale_domain_record" "glop" {
  domain = "${exoscale_domain.exo.id}"
  name = "glap"
  record_type = "CNAME"
  content = "${exoscale_domain.exo.name}"
}
```

Attributes:

- **`name`**: domain name

Values:

- `token`
- `state`
- `auto_renew`
- `expires_on`

Record attributes:

- **`domain`**: domain it's linked to
- **`name`**: name of the DNS record
- **`record_type`**: type of the DNS record. E.g. `A`, `CNAME`, `MX`, etc.
- **`content`**: value of the DNS record
- `ttl`: time to live
- `prio`: priority


### Storage on S3

```hcl
terraform = {
  backend "s3" {
    bucket = "..."
    endpoint = "https://sos-ch-dk-2.exo.io"
    key = "..."
    region = "ch-dk-2"
    access_key = "..."
    secret_key = "..."

    # Deactivate the AWS specific behaviours
    #
    # https://www.terraform.io/docs/backends/types/s3.html#skip_credentials_validation
    skip_credentials_validation = true
    skip_get_ec2_platforms = true
    skip_requesting_account_id = true
    skip_metadata_api_check = true
    skip_region_validation = true
  }
}
```

## Building

```
$ git clone https://github.com/exoscale/terraform-provider-exoscale
$ cd terraform-provider-exoscale
$ make build

# making a release (for Exoscale staff)
$ make release
```

### Development
```
# quick build of the provider
$ make

# updating the dependencies
$ make deps-update
```
