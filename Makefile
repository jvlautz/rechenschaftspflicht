.PHONY: dev
dev: .env
	@set -o allexport; . ./.env; set +o allexport; hivemind

.PHONY: dummy-data
dummy-data:
	rm -rf src/data/state.db
	python dummy-data-init.py

.env:
	cp .env.example .env

.PHONY: prod
prod: build-container
	docker compose up --build

.PHONY: build-container
build-container:
	docker build -t rechenschaftspflicht .

LOCAL_TRUNK_VER := $(shell date --utc +%Y%m%d%H%M%S)-$(shell git rev-parse --short HEAD)$(shell if git diff --quiet; then echo ""; else echo "-dirty"; fi)

.PHONY: release
release: build-container
	docker tag rechenschaftspflicht rknt/rechenschaftspflicht:$(LOCAL_TRUNK_VER)
	docker push rknt/rechenschaftspflicht:$(LOCAL_TRUNK_VER)
	cd /home/hff/repos/rknt-server && git pull
	yq -i '.services.rechenschaftspflicht.image = "rknt/rechenschaftspflicht:$(LOCAL_TRUNK_VER)"' /home/hff/repos/rknt-server/rechenschaftspflicht/docker-compose.yml
	cd /home/hff/repos/rknt-server && git add rechenschaftspflicht/docker-compose.yml && git commit -m "bump rechenschaftspflicht image version" && git push

.PHONY: check
check:
	cd src && go build ./...
	cd src && go vet ./...
	cd src && golangci-lint run ./...

.PHONY: fix
fix:
	cd src && go fmt ./...
	cd src && go fix ./...

.PHONY: test
test:
	cd src && go test ./...

.PHONY: integration
integration:
	cd src && go test -tags=integration -run TestIntegrationHappyPath -v .
