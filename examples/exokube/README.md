# Exokube single machine setup

```console
$ terraform apply

Outputs:

exokube_https = https://159.100.251.241.xip.io
exokube_ssh = ubuntu@159.100.251.241

$ ssh ubuntu@159.100.251.241
```

We have to wait for cloud-init to complete

```
exokube $ tail -f /var/log/cloud-init-output.log

Cloud-init v. 17.1 finished at ...
```

By downloading the `admin.conf`, which is a bad idea[™](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#optional-controlling-your-cluster-from-machines-other-than-the-master), you may administrate the remote server locally.

```
$ ssh ubuntu@159.100.251.241:.kube/config kubeconfig

$ kubectl --kubeconfig kubeconfig get nodes -o wide
NAME      STATUS     ROLES    AGE     VERSION   INTERNAL-IP       EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION      CONTAINER-RUNTIME
exokube   Ready      master   2m58s   v1.13.3   159.100.251.241   <none>        Ubuntu 16.04.5 LTS   4.4.0-116-generic   docker://18.6.2
```

## Running a Service

Let's run the 2048 game packaged by [**@sebgoa**](https://github.com/sebgoa/) as a docker image.

```
$ kubectl --kubeconfig kubeconfig run game --image=runseb/2048
deployment.apps/game created

$ kubectl --kubeconfig kubeconfig expose deployments game --port 80 --type NodePort
service/game exposed

$ kubectl --kubeconfig kubeconfig get svc game
NAME   TYPE       CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
game   NodePort   10.105.147.251   <none>        80:30917/TCP   4s
```

Then visit the service via its NodePort.

<http://159.100.251.241:30917/>

![screenshot of the game 2048](./2048.png)

## Setting up the Dashboard

[Web UI (Dashboard)](https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/)

```
$ kubectl --kubeconfig kubeconfig apply -f \
        https://raw.githubusercontent.com/kubernetes/dashboard/v1.10.1/src/deploy/recommended/kubernetes-dashboard.yaml

secret/kubernetes-dashboard-certs created
serviceaccount/kubernetes-dashboard created
role.rbac.authorization.k8s.io/kubernetes-dashboard-minimal created
rolebinding.rbac.authorization.k8s.io/kubernetes-dashboard-minimal created
deployment.apps/kubernetes-dashboard created
service/kubernetes-dashboard created
```


## Access the dashboard

Do not expose the dashboard using the NodePort technic, use the proxy instead.

```
$ kubectl --kubeconfig kubeconfig proxy
Starting to server on 127.0.0.1:8001
```

<http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/>

But you won't be able to log in, yet.

### Login using a token

To log into the dashboard, you need to authenticate as somebody or something (aka a Service Account).

```
exokube $ kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: admin-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: kubernetes-dashboard
  namespace: kube-system
EOF

exokube $ kubectl -n kube-system describe secrets \
   `kubectl -n kube-system get secrets | awk '/kubernetes-dashboard/ {print $1}'` \
       | awk '/token:/ {print $2}'
```

This is the token that may be used to log into the dashboard.

![screenshot of the dashboard](./dashboard.png)

[via this GitHub issue](https://github.com/kubernetes/dashboard/issues/2474#issuecomment-437815875)
