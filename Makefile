
.PHONY: compose build publish helm-template

compose:
	docker compose up --build

build:
	docker build . -f Dockerfile.publisher -t broswen/pg-publisher:latest

publish: build
	docker push broswen/pg-publisher:latest

helm-template:
	helm template publisher k8s/publisher > k8s/publisher.yaml

test: helm-template
	go test ./...
	kubeconform -summary -strict ./k8s