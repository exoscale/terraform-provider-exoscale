#cloud-config

write_files:
  - path: /etc/netplan/51-eip.yaml
    content: |
      network:
        version: 2
        renderer: networkd
        ethernets:
          lo:
            match:
              name: lo
            addresses:
            - ${eip}/32

runcmd:
  - ["netplan", "apply"]
