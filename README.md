# Building

Set ```GOPATH``` to this directory, and then run ```make```

Ensure that the PATH is set to include the resulting bin directory,
and then you can run the terraform command that will produce the
exoscale plugin.

Once built, you can install the terraform-provider-exoscale plugin by copying
the resulting  binary file into the location where the remaining Terraform
program and plugins reside.

# Terraform Usage

What follows below is the usage instructions for fully utilizing the Exoscale
resource plugin.  Additional documentation can be found in the examples directory.

## Provider requirements
```terraform
provider "exoscale" {
    token = ""
    secret = ""
    timeout = 60
}

```
You are required to provide at least the OAuth API token and secret key in order
to make use of the remaining Terraform resources.

The `timeout` is the maximum amount of time (in seconds, default: `60`) to wait for async tasks to complete. Currently, this is used
during the creation of `compute` and `anti-affinity` resources.

You can specify the environment variables for these using ```EXOSCALE_API_SECRET```
or ```EXOSCALE_API_KEY```.  You can also use the cloudstack environment variables
`CLOUDSTACK_(API|SECRET)_KEY`.

## SSH Resource

Declare an ssh key that will be used for any current/future instances

```terraform
resource "exoscale_ssh" "keylabel" {
    name = "keyname"
    key = "keycontents"
}
```

* ```name``` Defines the label in Exoscale to define the key
* ```key``` The ssh public key that will be copied into instances declared

## Anti-Affinity Groups

Define an affinity group that can be used to group various instances together

```terraform
resource "exoscale_affinity" "affinitylabel" {
    name = "affinity name"
}
```

* ```name``` Defines the affinity label that will be used by other declared instances

## Security Groups

Provide a named grouping of firewall rules that would be applicable for each
instance.

```terraform
resource "exoscale_securitygroup" "sglabel" {
    name = "sgname"
    ingress_rules = {
      cidr = "0.0.0.0/0"
      protocol = "TCP"
      port = 22
    }
    egress_rules = {
      cider = "192.168.1.0/24"
      protocol = "TCP"
      port = 22
    }
    egress_rules = {
      cidr = "192.168.1.0/24"
      protocol = "ICMP"
      icmptype = 0
      icmpcode = 0
    }
}
```

* ```name``` Security Group name as it will be referenced in the instances
* ```ingress_rules``` One or more rules to describe which ports will be permitted inbound
 * ```cidr``` A network address range to reflect who would be impacted
 * ```protocol``` Indicate the type to look for TCP, UDP, or ICMP
 * ```port``` For TCP/UDP the port number of the service impacted
 * ```icmptype``` ICMP message type
 * ```icmpcode``` ICMP message code
* ```egress_rules``` One or more rules to describe which ports will be permitted outbound
 * ```cidr``` A network address range to reflect who would be impacted
 * ```protocol``` Indicate the type to look for TCP, UDP, or ICMP
 * ```port``` For TCP/UDP the port number of the service impacted
 * ```icmptype``` ICMP message type
 * ```icmpcode``` ICMP message code

## Compute Instances

Define a new compute resource.

```terraform
resource "exoscale_compute" "computelabel" {
    name = "testname"
    template = "ubuntu-16.04"
    zone = "ch-gva-2"
    size = "Micro"
    disk_size = 10
    keypair = "terraformKey"
    affinitygroups = ["terraformag"]
    securitygroups = ["sshgroup"]
    userdata = ""
}
```

* ```name``` The compute resource hostname
* ```template``` The template to use for the specified resource
* ```size``` Defines the instance configuration size:
 * Micro
 * Tiny
 * Small
 * Medium
 * Large
 * Extra-Large
 * Huge
* ```disk_size``` Define the size of the root disk: 10GB, 50GB, 100GB, 200GB, 400GB
* ```zone``` One of the two datacenters: CH-DK-2 and CH-GVA-2
* ```keypair``` The SSH key used for root access to the host
* ```affinitygroups``` Collection of anti-affinity groups the host will belong to
* ```securitygroups``` Collection of security groups to indicate which rules will apply
* ```userdata``` Free form statements used for configuring the instance

## DNS

If the user has an active DNS subscription with Exoscale, allow them the ability
to manage their DNS information.

```terraform
resource "exoscale_dns" "testdomain" {
    name = "testdomain.ch"
    record = {
        name = "test1"
        type = "A"
        content = "192.168.1.1"
    }
    record = {
        name = "test2"
        type = "CNAME"
        content = "test1"
    }
}
```

* ```name``` The domain name to be managed
* ```record``` Collection of records to be included as a part of the name
 * ```name``` The host name to define the record
 * ```type``` The DNS entry type such as the CNAME, MX, or A
 * ```content``` The requisite component for the corresponding record name and type
 * ```ttl``` Optional time to live for the record
 * ```prio``` Optional record priority

## S3

There are two resources that define the S3 interaction: buckets for the
creation/management of the bucket name, and objects for the contents of said
buckets.

```terraform
resource "exoscale_s3bucket" "testbucket" {
    bucket = "tftest"
    acl = "private"
}
```

* ```bucket``` The bucket name that will be referenced in all object references
* ```acl``` Permission type for the bucket and its contents based off the AWS S3 implementation

```terraform
resource "exoscale_s3object" "testobj" {
    bucket = "tftest"
    acl = "private"
    key "test/path.txt"
    type = "text/plain"
    content = "hello world"
}

resource "exoscale_s3object" "testobj" {
    bucket = "tftest"
    acl = "private"
    key "test/path2.txt"
    type = "text/plain"
    source = "/tmp/test.txt"
}
```

* ```bucket``` The bucket the object will be contained under
* ```acl``` Permission type for the bucket and its contents based off the AWS S3 implementation
* ```key``` A directory/file path used to reference the object as its key
* ```type``` A mime type to indicate the type of file
* ```content``` Something that can be injected directly into the bucket at the key
* ```source``` The path to a file that will be uploaded into the bucket at the key

While content and source are mutually exclusive, one of them is required for the
operation to succeed.

# TODO List/Missing features

## Security Groups
* Support single port declaration as well as starting/ending port ranges

## S3 Support
* Due to the AWS library in use, CORS is not supported
* Due to the AWS library in use, per-object K/V pairs are not supported
