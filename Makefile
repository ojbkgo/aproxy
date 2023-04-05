image_name=registry.cn-hangzhou.aliyuncs.com/meeya/proxy:v1.0.0

all: build package

build-local:
	go build -o bin/proxy ./cmd
	#go build -o bin/linux_amd64/proxy ./cmd
	#chmod 0777 bin/linux_amd64/proxy

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/linux_amd64/proxy ./cmd
	chmod 0777 bin/linux_amd64/proxy

package:
	docker build -t ${image_name} .
	docker push ${image_name}

debug: build-local build
	scp bin/linux_amd64/proxy root@129.211.212.74:/root/proxy
