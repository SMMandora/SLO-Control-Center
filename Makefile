COMPOSE = docker compose -f deploy/compose/docker-compose.yml
GOBIN   = $(shell go env GOPATH)/bin
K8S_SVCS = orders-api payments-worker notification-svc mock-gateway mock-receiver slo-bff alert-receiver

.PHONY: up down logs test dashboards k8s-up k8s-down k8s-images deploy

up: ## Build + start the full stack
	$(COMPOSE) up --build -d

down: ## Stop the stack and remove volumes
	$(COMPOSE) down -v

logs: ## Tail all service logs
	$(COMPOSE) logs -f

test: ## Run all unit tests (Go services, Python, frontend, Prometheus rules)
	cd services/orders-api && go test ./...
	cd services/slo-bff && go test ./...
	cd services/notification-svc && go test ./...
	cd services/mock-gateway && go test ./...
	cd services/alert-receiver && go test ./...
	cd services/payments-worker && python -m pytest -q
	cd frontend && npx vitest run
	docker run --rm --entrypoint promtool \
		-v "$(CURDIR)/observability/prometheus/rules:/rules" -w /rules \
		prom/prometheus:latest test rules orders-api.slo.rules.test.yml mesh.slo.rules.test.yml alerts.test.yml

k8s-up: ## Create the local kind cluster
	$(GOBIN)/kind create cluster --config deploy/k8s/kind-cluster.yaml

k8s-down: ## Delete the kind cluster
	$(GOBIN)/kind delete cluster --name slo

k8s-images: ## Build service images and load them into kind
	for s in $(K8S_SVCS); do docker build -t slo/$$s:dev services/$$s; done
	docker build -t slo/frontend:dev frontend
	$(GOBIN)/kind load docker-image --name slo \
		$(foreach s,$(K8S_SVCS),slo/$(s):dev) slo/frontend:dev

deploy: ## Apply the staging overlay to the kind cluster (run k8s-images first)
	$(GOBIN)/kustomize build --load-restrictor LoadRestrictionsNone deploy/k8s/overlays/staging | kubectl apply -f -

dashboards: ## Regenerate Grafana dashboards from Grafonnet source
	cd observability/grafana/jsonnet && PATH="$(PATH):$(GOBIN)" jb install
	cd observability/grafana/jsonnet && PATH="$(PATH):$(GOBIN)" \
		jsonnet -J vendor slo-overview.jsonnet > ../dashboards/slo-overview.json
	cd observability/grafana/jsonnet && PATH="$(PATH):$(GOBIN)" \
		jsonnet -J vendor incident.jsonnet > ../dashboards/incident-investigation.json
	cd observability/grafana/jsonnet && PATH="$(PATH):$(GOBIN)" \
		jsonnet -J vendor service-drilldown.jsonnet > ../dashboards/service-drilldown.json
	cd observability/grafana/jsonnet && PATH="$(PATH):$(GOBIN)" \
		jsonnet -J vendor capacity-use.jsonnet > ../dashboards/capacity-use.json
