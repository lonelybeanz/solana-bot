ORGANIZATION = "organization"
APP = "bin/solana-bot"


.PHONY: build
build:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags no_k8s -ldflags "-s -w " -o $(APP) main.go


.PHONY: deploy

DEPLOY_SCRIPT ?= deploy.sh  # 默认执行 deploy.sh

deploy: build
	./scripts/$(DEPLOY_SCRIPT)