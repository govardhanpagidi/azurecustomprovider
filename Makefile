goos=linux
goarch=amd64

build:
	env GOOS=$(goos) GOARCH=$(goarch) go build -o handler