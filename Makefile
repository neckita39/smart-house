.PHONY: build web test run clean

# Собрать фронтенд и бинарник.
build: web
	go build -o smart-house .

# Собрать фронтенд в web/dist (нужен Node).
web:
	cd web && npm install && npm run build

test:
	gofmt -l . && go vet ./... && go test ./...

run:
	go run .

clean:
	rm -rf smart-house web/dist
