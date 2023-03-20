image_name=registry.cn-hangzhou.aliyuncs.com/meeya/proxy:latest

all: build package

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/linux_amd64/proxy-agent ./cmd/proxy-agent
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/linux_amd64/proxy-gateway ./cmd/proxy-gateway

package:
	docker build -t ${image_name} .
	docker push ${image_name}