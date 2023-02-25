.PHONY: compose build

compose:
	docker compose up --build

build:
	docker build . -f Dockerfile.publisher -t pg-publisher:latest
