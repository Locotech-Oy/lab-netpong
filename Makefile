DOCKER_IMAGE=netpong:local

.PHONY: all_k8s_local docker_build k8s_local

all_k8s_local: docker_build k8s_local_run

build:
	go build main.go

docker_build:
	docker build -t ${DOCKER_IMAGE} .

k8s_local_clean:
	-kubectl delete -f k8s/netpong.yml

k8s_local_run: k8s_local_clean
	kubectl apply -f k8s/netpong.yml