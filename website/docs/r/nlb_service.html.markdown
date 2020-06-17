---
layout: "exoscale"
page_title: "Exoscale: exoscale_nlb_service"
sidebar_current: "docs-exoscale-nlb-service"
description: |-
  Provides an Exoscale Network Load Balancer service resource.
---

# exoscale\_nlb\_service

Provides an Exoscale Network Load Balancer ([NLB][nlb]) service resource. This can be used to create, modify, and delete NLB services.

[nlb]: nlb.html

## Example Usage

```hcl
variable "zone" {
  default = "de-fra-1"
}

variable "template" {
  default = "Linux Ubuntu 20.04 LTS 64-bit"
}

data "exoscale_compute_template" "website" {
  zone = var.zone
  name = var.template
}

resource "exoscale_instance_pool" "website" {
  name = "instancepool-website"
  description = "Instance Pool Website nodes"
  template_id = data.exoscale_compute_template.website.id
  service_offering = "medium"
  size = 3
  zone = var.zone
}

resource "exoscale_nlb" "website" {
  name = "website"
  description = "This is the Network Load Balancer for my website"
  zone = var.zone
}

resource "exoscale_nlb_service" "website" {
  zone = exoscale_nlb.website.zone
  name = "website"
  description = "Website over HTTP"
  nlb_id = exoscale_nlb.website.id
  instance_pool_id = exoscale_instance_pool.website.id
	protocol = "tcp"
	port = 80
	target_port = 8080
	strategy = "round-robin"

  healthcheck {
    port = 8080
    mode = "http"
    uri = "/healthz"
    interval = 5
    timeout = 3
    retries = 1
  }
}
```

## Argument Reference

* `nlb_id` - (Required) The ID of the NLB to attach the service.
* `zone` - (Required) The name of the [zone][zone] used by the NLB.
* `instance_pool_id` - (Required) The ID of the Instance Pool to forward network traffic to.
* `name` - (Required) The name of the NLB service.
* `port` - (Required) The port of the NLB service.
* `target_port` - (Required) The port to forward network traffic to on target instances.
* `protocol` - The protocol (tcp/udp).
* `strategy` - The strategy (round-robin/source-hash).
* `description` - The description of the NLB service.

**healthcheck**

* `port` - (Required) The healthcheck port.
* `mode` - The healthcheck mode (tcp/http).
* `uri` - The healthcheck URI, must be set only if `mode` is `http`.
* `interval` - The healthcheck interval in seconds.
* `timeout` - The healthcheck timeout in seconds.
* `retries` - The healthcheck retries.

[zone]: https://www.exoscale.com/datacenters/

## Import

An existing NLB service can be imported as a resource by ID. Importing a NLB service imports the `exoscale_nlb_service` resource.

```console
$ terraform import exoscale_nlb_service.website 9ecc6b8b-73d4-4211-8ced-f7f29bb79524
```
