## 0.11.0 (Unreleased)

FEATURES:

- **New Data Source:** `exoscale_compute_template` (#231)

IMPROVEMENTS:

- Add support for *managed* Elastic IP to the `exoscale_ipaddress` resource

## 0.10.0 (March 6, 2019)

- dep: playing with terraform v0.12.0-beta1 (#200)

## 0.10.0-beta1 (March 4, 2019)

- examples: fix syntax
- terraform 0.12-beta1

## 0.9.46 (March 4, 2019)

- affinity: fix virtual machine ids (#220)

## 0.9.45 (February 28, 2019)

- dep: egoscale v0.14.3 (#219)
- rules: fix egress update (#218)
- examples: k8s using kubeadm (#67)
- Ignore drift for object_lock_configuration (#216)
- website: build locally using middleman (#214)

## 0.9.44 (February 21, 2019)

- Add CAA to domainRecordResource (#215)

## 0.9.43 (February 12, 2019)

- security group: fix import (#212)

## 0.9.42 (February 8, 2019)

- rules: allow creating a batch of ingress/egress rules (#199)
- mod: pretend this project is already part of terraform-providers (#209)

## 0.9.41 (January 10, 2019)

- compute: keep base64 encoded user_data as is (#206)
- project: upgrade terraform v0.11.11 (#204)
- examples: add managed privnet (#203)

## 0.9.40 (December 12, 2018)

- test: fix the acceptance tests
- exoscale: adapt to library changes
- vendor: bump libraries

## 0.9.39 (November 19, 2018)

- dns record: fail (#202)

## 0.9.38 (November 13, 2018)

- secondary ip: fix id (#201)
- no domain (#198)

## 0.9.37.1 (November 2, 2018)

- network: remove cidr (#197)

## 0.9.37 (November 2, 2018)

This release features the managed privnet (DHCP) capabilities, only in the `ch-gva-2` zone for the time being.

- travis: copy AWS provider travis setup (#193)
- managed privnet: the code (#186)
- sg rule: import using only the ruleid (#190)
- security group rule: add IPIP (#191)
- Fix README's reference to the CloudStack configuration file (#189)
- security group: no more tags (#180)

## 0.9.36 (August 31, 2018)

- compute: fix #181
- examples: remove tags on security groups (#178)

## 0.9.35 (August 29, 2018)

- update deps (#177)
- Dep updates (#175)

## 0.9.34 (August 16, 2018)

- egoscale v0.11 (#173)
- tests: Arftul is no more (#172)

## 0.9.33 (August 3, 2018)

- provider: http traces (#170)
- Update ego (#171)
- dep: update egoscale (and others) (#168)

## 0.9.32 (July 19, 2018)

- dep: bump go-ini version
- validation: adding tests (and fixing bugs) (#162)
- security group: test updating the tags (#165)
- test: updating compute instance (#166)
- compute: don't udpate size if they virtually are the same (#164)
- security group rule: add acceptance test (#159)
- domain record: add acceptance test (#161)
- Add port 10250 as it is prerequesite (#160)

## 0.9.31 (July 17, 2018)

- compute: less validation to enable GPU SO (#157)
- secondary ip: add acceptance test (#156)
- provider: more envs (#155)
- nic: add acceptance test (#154)
- Network acc (#153)
- deps: use less-types branch (#121)

## 0.9.30 (July 6, 2018)

- dep: update go-ini to 1.37
- domain: try to not erase things (#150)
- secondary ipaddress: fix import (#149)
- dep: upgrade egoscale to 0.9.31
- global: bump default timeout to 5m (#152)
- travis: run acceptance test on travis (#148)

## 0.9.29 (June 28, 2018)

- dep: bump egoscale to 0.9.30 (#146)
- examples: creating a bucket using aws provider (#104)

## 0.9.28 (June 26, 2018)

- import DNS record (#144)
- provider: when cloudstack.ini is used, build dns_endpoint (#128)
- dep: ensure -update

## 0.9.27 (June 21, 2018)

- Possibility to disable gzipping user-data (#142)

## 0.9.26 (June 20, 2018)

- `security_group_rule`: handle gone security group (#141)

## 0.9.25 (June 19, 2018)

- put hashicorp's gitignore
- secondary ip: create a compound id instead of the cs id (#126)
- fixup! global: use hashi's scripts
- readme: use hashi's readme (#140)
- license: change to MPL2 (#139)
- global: add changelog for hashi'
- global: use hashi's scripts
- global: rename files according hashi's conventions
- examples: add RKE example
- compute: use Details to activate ipv6
- Update ipaddress.html.markdown

## 0.9.24 (June 4, 2018)

- provider: better error message
- provider: fix another nil
- goreleaser: disable CGO

## 0.9.23 (June 4, 2018)

- goreleaser: fix binary name
- fix: nil pointer check
- deps: upgrade egoscale to 0.9.27
- build: cleanup makefile
- build. use goreleaser
- tests: add acceptance test for DNS domain
- examples: add DNS example

## 0.9.22 (May 15, 2018)

- `exoscale_compute.user_data` is know read from the external resource
- allow `ALL` protocol rule
- documentation fixes, thanks to @mcorbin (#116) 
- upgrade egoscale to 0.9.25

## 0.9.21 (April 27, 2018)

IMPROVEMENTS:

- Upgrade egoscale to 0.9.22
- Upgrade terraform to 0.11.7

BUG FIXES:

- Fix example in documentation

## 0.9.20 (April 20, 2018)

IMPROVEMENTS:

- Allow `user_data` to be updated, #113 
- Upgrade egoscale to 0.9.21

## 0.9.19 (April 13, 2018)

IMPROVEMENTS:

- Read SSH `username` from the template details (#111)
- Upgraded egoscale to 0.9.20

## 0.9.18 (March 28, 2018)

BUG FIXES:

- `compute` resource `ipv6` attribute wasn't properly set (#107)

## 0.9.17 (March 27, 2018)

IMPROVEMENTS:

- Upgrade egoscale version

## 0.9.16 (March 27, 2018)

BUG FIXES:

- `security_group_rule` may start at zero
- Compute `state="Stopped"` wasn't applied

## 0.9.15 (March 23, 2018)

IMPROVEMENTS:

- IPv6 for `compute` resources
- `ICMPv6` and `/128` CIDR for `security_group_rule` resources

BUG FIXES:

- fix: tags weren't set after creation
- fix: crash during import (nil pointer)

## 0.9.14 (March 20, 2018)

IMPROVEMENTS:

- A SSH key pair may be created
- Support timeouts on every call
- `compute` resource can retrieve the password and encrypted password

BUG FIXES:

- Error message on 40x responses

## 0.9.13 (March 13, 2018)

DEPRECATIONS:

- Use `key` instead of `token`

IMPROVEMENTS:

- `exoscale_compute` has separate `affinity_groups`/`affinity_group_names` and `security_groups`/`security_group_ids`

BUG FIXES:

- Handle missing Elastic IP when doing the import

## 0.9.12 (March 2, 2018)

IMPROVEMENTS:

- `exoscale_domain_record` offers a `hostname` field. Handy for `CNAME` records.

## 0.9.11 (March 2, 2018)

IMPROVEMENTS:

- A `compute` resource can be deployed without being started (#68)

BUG FIXES:

- Less `<nil>` values in `nic` and `network` resources
- `user_data` and `key_pair` force the creation of a new `compute` resource
- The `~/.cloudstack.ini` file is read by default

## 0.9.10 (March 1, 2018)

IMPROVEMENTS:

- Importing a `compute` resource will also import any `secondary_ipaddress` resource linked to it

CHANGES:

- Separate `user_security_group` and `user_security_group_id` within a `security_group_rule`

BUG FIXES:

- Importing a missing network failed silently
- Auto fill the `security_groups` and `affinity_groups` of a `compute` resource

## 0.9.9 (February 1, 2018)

BUG FIXES:

- Importing a missing compute fails
- Secret/token conflicts with config/provider
- Updating a compute crashes
- Network refreshes the tags

## 0.9.8 (February 1, 2018)

IMPROVEMENTS:

- A `compute` resource can be imported by its name
- `cloudstack.ini` files are supported
- Tags are supported on Security Groups, Networks and Elastic IP

## 0.9.7 (January 29, 2018)

BUG FIXES:

- _nil pointer_ error when working on missing resources
- `user_data` is auto-magically encoded in base64 without having to use `template_cloudinit_config`

## 0.9.6 (January 22, 2018)

IMPROVEMENTS:

- A Security Group may be imported using its name as well as its ID

BUG FIXES:

- Global variables documentation didn't match the actual code (#49)
- Domain record missing content field (#50)
- Importing security group rule misses `user_security_group` key (#51)

## 0.9.5 (January 19, 2018)

IMPROVEMENTS:

- Updated egoscale
- Added example cloud-init multi-part setup

BUG FIXES:

- bug fix `IP` addresses

## 0.9.4 (January 18, 2018)

FEATURES:

- `exoscale_network` and `exoscale_nic` for [multiple private networks](https://www.exoscale.ch/syslog/2018/01/17/introducing-multiple-private-networks/)

IMPROVEMENTS:

- Examples are fresh
- Using godep for managing dependencies

## 0.9.3 (January 15, 2018)

IMPROVEMENTS:

- `exoscale_affinity_group` shows the list of machines that are part of the group

BUG FIXES:

- Refreshing a resource that was deleted via the console
- Security Group rule `cidr` key

## 0.9.2 (January 11, 2018)

FEATURES:

- **New Resource:** `exoscale_domain_record`
- **New Resource:** `exoscale_security_group_rule`
- **New Resource:** `exoscale_ipaddress`

IMPROVEMENTS:

- `exoscale_compute` with import
- `exoscale_ssh_keypair` with import
- `exoscale_affinity` with import
- `exoscale_domain` with import containing the DNS records
- `exoscale_security_group` with import containing the _Security Group_ rules
- `exoscale_secondary_ipaddress` associate a compute and an elastic IP address

NOTES:

The following features are missing/unstable:

- Tags only on `compute`
- S3 bucket and objects

BREAKING CHANGES:

This version is mostly not backward compatible with the previous release of the provider.

## 0.1.0 (December 11, 2017)

First release