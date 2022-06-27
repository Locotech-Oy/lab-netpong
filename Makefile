DOCKER_IMAGE=netpong
DOCKER_TAG=local

.PHONY: all_k8s_local docker_build k8s_local

all_k8s_local: docker_build k8s_local_run

build:
	go build main.go

docker_build:
	docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} .

k8s_local_clean:
	-kubectl delete -f k8s/netpong.yml

k8s_local_run: k8s_local_clean
	kubectl apply -f k8s/netpong.yml

gcp_deploy_cluster:
	terraform -chdir=terraform/gcp init
	terraform -chdir=terraform/gcp apply

gcp_destroy_cluster:
	terraform -chdir=terraform/gcp destroy