name="aibank-ms"

build:
	go build
	./cktool -path "/Users/kevintung/code/$(name)" -branch "$(branch)"

build_param:
	go build 
	./cktool -path "/Users/kevintung/code/$(name)" -start $(start) -end $(end) -tag-count $(count)

build_count:
	go build 
	./cktool -path "/Users/kevintung/code/$(name)" -tag-count $(count)

build_target_linux:
	GOOS=linux GOARCH=amd64 go build -o cktool main.go

build_target_darwin:
	GOOS=darwin GOARCH=arm64 go build -o cktool main.go

build_all:
	if [[ ! -d ./macos ]]; then mkdir macos; fi
	if [[ ! -d ./windows ]]; then mkdir windows; fi
	if [[ ! -d ./linux ]]; then mkdir linux; fi
	GOOS=linux GOARCH=amd64 go build -o cktool main.go && mv ./cktool ./linux/
	GOOS=darwin GOARCH=arm64 go build -o cktool main.go && mv ./cktool ./macos/
	GOOS=windows GOARCH=amd64 go build -o cktool main.go && mv ./cktool ./windows/


