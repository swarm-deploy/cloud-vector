lint:
	golangci-lint run

build:
	docker build . -t wmb-prod.cr.cloud.ru/infra/monitoring/cloud-vector:latest && \
	docker push wmb-prod.cr.cloud.ru/infra/monitoring/cloud-vector:latest