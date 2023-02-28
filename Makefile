.PHONY: compose build

compose:
	docker compose up --build

build:
	docker build . -f Dockerfile.publisher -t pg-publisher:latest


helm-template:
	helm template pg-publisher k8s/pg-publisher > k8s/pg-publisher.yaml