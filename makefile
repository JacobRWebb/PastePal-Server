run:
	go mod tidy
	go run cmd/pastepal/main.go

build:
	go mod tidy
	go build -o pastepal.exe cmd/pastepal/main.go

