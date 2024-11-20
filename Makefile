.PHONY: build build-linux deploy clean

build: out
	GOOS=darwin GOARCH=arm64 go build -o ./out/lyon-transports-darwin-arm64 ./

build-linux: out
	GOOS=linux GOARCH=amd64 go build -o ./out/lyon-transports-linux-amd64 ./

deploy: build-linux
	ssh ubuntu "sudo systemctl stop lyon-transports.service"
	scp ./out/lyon-transports-linux-amd64 ubuntu:/mnt/data/lyon-transports/
	ssh ubuntu "sudo systemctl start lyon-transports.service"

out:
	mkdir -p out

clean:
	rm -rf out
