build: cmd/arbi/main.go
	go build -o arbi cmd/arbi/main.go
test:
	go test ./...
