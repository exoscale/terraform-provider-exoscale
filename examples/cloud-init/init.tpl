#cloud-config

manage_etc_hosts: true
fqdn: ${fqdn}

apt_sources:
- source: "deb [arch=amd64] https://download.docker.com/linux/ubuntu ${ubuntu} stable"
  keyid: 9DC858229FC7DD38854AE2D88D81803C0EBFCD88

package_update: true
package_upgrade: true

packages:
- docker-ce
