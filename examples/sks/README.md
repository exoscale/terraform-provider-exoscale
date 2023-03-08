# Scalable Kubernetes Service (SKS)

This example demonstrates how to instantiate a
[Scalable Kubernetes Service (SKS) Cluster](https://community.exoscale.com/documentation/sks/),
using the `exoscale_sks` and `exoscale_sks_nodepool` resource.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

One should note the adminitration credentials - `kubeconfig` - being dumped locally thanks to the
`exoscale_sks_kubeconfig` and `local_sensitive_file` resources.

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

my_sks_cluster_endpoint = "https://791cb2bc-b7d9-4a19-92ac-20edd9165458.sks-ch-gva-2.exo.io"
my_sks_connection = "export KUBECONFIG=kubeconfig; kubectl cluster-info; kubectl get pods -A"
my_sks_kubeconfig = "kubeconfig"

$ export KUBECONFIG=kubeconfig; kubectl cluster-info; kubectl get pods -A
Kubernetes control plane is running at https://791cb2bc-b7d9-4a19-92ac-20edd9165458.sks-ch-gva-2.exo.io:443
CoreDNS is running at https://791cb2bc-b7d9-4a19-92ac-20edd9165458.sks-ch-gva-2.exo.io:443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
NAMESPACE     NAME                                       READY   STATUS    RESTARTS   AGE
kube-system   calico-kube-controllers-56cdb7c587-tbbqq   1/1     Running   0          7m35s
kube-system   calico-node-btt9w                          1/1     Running   0          6m26s
kube-system   calico-node-glk44                          1/1     Running   0          6m35s
kube-system   calico-node-r58b9                          1/1     Running   0          6m30s
kube-system   coredns-648647979-7z4tb                    1/1     Running   0          7m28s
kube-system   coredns-648647979-p6ljn                    1/1     Running   0          7m28s
kube-system   konnectivity-agent-dd97df547-f55pc         1/1     Running   0          7m27s
kube-system   konnectivity-agent-dd97df547-gtfpq         1/1     Running   0          7m27s
kube-system   kube-proxy-42wbv                           1/1     Running   0          6m26s
kube-system   kube-proxy-cfkmc                           1/1     Running   0          6m35s
kube-system   kube-proxy-pdgpd                           1/1     Running   0          6m30s
kube-system   metrics-server-77b474bd7b-68c75            1/1     Running   0          7m25s
```
