.PHONY: build web test run clean build-intel deploy

# Собрать фронтенд и бинарник.
build: web
	go build -o smart-house .

# Собрать фронтенд в web/dist (нужен Node); emptyOutDir выключен (чтобы go:embed
# не ломался на пустом каталоге в чистом клоне), поэтому чистим старый бандл сами.
web:
	rm -rf web/dist/assets web/dist/index.html
	cd web && npm ci && npm run build

test:
	test -z "$$(gofmt -l .)" && go vet ./... && go test ./...

run:
	go run .

clean:
	rm -rf smart-house web/dist build

# Кросс-сборка под Intel Mac (домашний сервер) — бинарник без CGO.
build-intel: web
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o build/smart-house-darwin-amd64 .

# Деплой на домашний сервер: make deploy DEPLOY=user@192.168.1.95
# .env не копируется автоматически — его нужно скопировать вручную один раз.
deploy: build-intel
ifndef DEPLOY
	$(error укажите DEPLOY=user@host, например make deploy DEPLOY=user@192.168.1.95)
endif
	rsync -az build/smart-house-darwin-amd64 $(DEPLOY):smart-house/smart-house
	rsync -az deploy/ $(DEPLOY):smart-house/deploy/
	@echo "скопируйте .env один раз: scp .env $(DEPLOY):smart-house/.env"
	ssh $(DEPLOY) 'cd smart-house && bash deploy/install.sh'
