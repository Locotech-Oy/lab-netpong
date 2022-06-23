# lab-netpong
A small kubernetes based setup that throws some traffic around in random ways. The setup is based around terraform and kubernetes.

## Working principle

When a pod starts, it will wait for a random nr of ms before attempting to send a packet to one of it's neighbours. Neighbours are determined by looking for other pods in the namespace having a prefix "netpong" (can be adjusted via flag). From the list of found pods, the current pod will choose one random neighbour and try to send a packet over the internal network. The receiving pod answers with an OK message, increases the numHops counter, then attempts to forward the packet to another neighbour. The pod will only attempt to create a new packet if it hasn't already received a packet from another pod during the time it started to the time it would do it's first sendout.

Once the packet has been sent and acknowledged by the recipient, the address of the recipient is stored in the senders local cache (in-memory). Whenever the pod receives a packet in the future, it will first attempt to forward it to the recipient in the cache. If the send for some reason fails, e.g. the pod does not exist anymore or has been blocked by a network policy, the sender will add that container to a "deadpool" list, and not attempt to send packets there again until the setup is restarted.

The packet structure is a json payload that contains:
{
  "numHops": 12, // number of containers this packet has passed through
  "hostList": []  // list of all containers this packet has passed through (hosts only add to the list if they do not already exist on it)
}

## Resources used
https://kubernetes.github.io/ingress-nginx/


## Build container

From with the root folder of the project (where the Dockerfile exists), run:
```
docker build -t netpong:local .
```

change netpong:local to another tag if you want to push to another repository

## Run in local kubernetes cluster

Login / switch context to your local cluster. Then run

```
kubectl apply -f k8s/netpong.yml
```

## Login to GCP

```
gcloud auth application-default login
```

authenticate kubectl for use with GCP cluster (this requires the cluster to be up and running, see below):
```
gcloud container clusters get-credentials netpong-cluster-prod --region=europe-north1
```


## Run in your environment

Note: this will incur costs, so tear down the environment again once you don't need it.

go to ```terraform/gcp``` folder
```
terraform init
terraform apply
```
Provide name of project you will host the cluster in. Terraform will spin up a GKE setup and output a kubeconfig file (suffix ```-prod```
 by default)

To start up a generic nginx-based ingress controller:

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.2.0/deploy/static/provider/cloud/deploy.yaml
```

check status with:
```
kubectl get pods -n ingress-nginx
kubectl get svc -n ingress-nginx
```

Check public IP of ingress controller:
```
kubectl get service ingress-nginx-controller --namespace=ingress-nginx
```
once public IP available, go into k8s folder and create test pods using

```
kubectl create -f nginx.yml
```

using the kubeconfig from the terraform folder. This will setup a Nginx ingress and a couple of pods.

## Enable Prisma Cloud integration

Download a DaemonSet from Prisma Cloud Console -> Compute -> Manage -> Deploy -> Defenders. For GKE, ensure the option "Nodes use Container Runtime Interface (CRI), not Docker." option is active. Deploy the defenders using

kubectl create -f daemonset.yml

## Tear down environment

go to terraform/gcp folder
terraform destroy

(assumes you've run terraform init/apply from this folder)

to clean up demo app
kubectl delete -f fanout-ingress.yml
kubectl delete -f web.yml

to clean up ingress controller:
kubectl delete all  --all -n ingress-nginx
