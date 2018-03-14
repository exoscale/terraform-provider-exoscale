# SSH Keys

Take a look at [main.tf](main.tf) for the real deal and configure
`cloudstack.ini` with your credentials.

```console
$ terraform apply
...
exoscale_compute.vm (remote-exec): Connecting to remote host via SSH...
exoscale_compute.vm (remote-exec):   Host: 185.150.8.15
exoscale_compute.vm (remote-exec):   User: ubuntu
exoscale_compute.vm (remote-exec):   Password: false
exoscale_compute.vm (remote-exec):   Private key: true
exoscale_compute.vm (remote-exec):   SSH Agent: false
exoscale_compute.vm (remote-exec): Connected!
exoscale_compute.vm (remote-exec): Linux myvm 4.13.0-36-generic #40-Ubuntu SMP Fri Feb 16 20:07:48 UTC 2018 x86_64 x86_64 x86_64 GNU/Linux
```

The recipe creates an SSH key, puts it into a machine and uses
the private key to connect into it and run a mere `uname -a`.
