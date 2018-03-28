#cloud-config
---
manage_etc_hosts: true
fqdn: ${fqdn}

package_update: true
package_upgrade: true

packages:
- jq
