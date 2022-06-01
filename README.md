# lab-netpong
A small kubernetes based setup that throws some traffic around in random ways. The setup is based around terraform and kubernetes.


## Resources used
https://kubernetes.github.io/ingress-nginx/


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

Go into k8s folder

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.2.0/deploy/static/provider/cloud/deploy.yaml
```

This will setup an ingress controller, check status with:
```
kubectl get pods -n ingress-nginx
kubectl get svc -n ingress-nginx
```

Check public IP of ingress controller:
```
kubectl get service ingress-nginx-controller --namespace=ingress-nginx
```
once public IP available, create test structures

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
