name="aibank-ms"

build:
	go build
	./cktool -path "/Users/kevintung/code/$(name)"

build_param:
	go build 
	./cktool -path "/Users/kevintung/code/$(name)" -start $(start) -end $(end)

build_target_linux:
	GOOS=linux GOARCH=amd64 go build -o cktool main.go

build_target_darwin:
	GOOS=darwin GOARCH=arm64 go build -o cktool main.go
