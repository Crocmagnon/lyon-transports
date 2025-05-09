.PHONY: build build-linux deploy clean

build: out
	GOOS=darwin GOARCH=arm64 go build -o ./out/lyon-transports-darwin-arm64 ./

build-linux: out
	GOOS=linux GOARCH=amd64 go build -o ./out/lyon-transports-linux-amd64 ./

deploy-ubuntu: build-linux
	ssh ubuntu "sudo systemctl stop lyon-transports.service"
	scp ./out/lyon-transports-linux-amd64 ubuntu:/mnt/data/lyon-transports/
	ssh ubuntu "sudo systemctl start lyon-transports.service"

deploy-vps: build-linux
	ssh vps-1 "sudo systemctl stop lyon-transports.service"
	scp ./out/lyon-transports-linux-amd64 vps-1:/home/ubuntu/deploy/lyon-transports/
	ssh vps-1 "sudo systemctl start lyon-transports.service"

out:
	mkdir -p out

clean:
	rm -rf out
