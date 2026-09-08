.PHONY: build web test run clean

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
	rm -rf smart-house web/dist
