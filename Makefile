.PHONY: clean deps simplify run test coverage build update-certs

clean:
		rm -rf target; \
		rm -f coverage.*

deps: clean
		go get -d -v ./...

simplify:
		gofmt -s -l -w .

run: deps
		go run *.go

test: deps
		go test -count=1 -v ./...

coverage: test
		go test -coverprofile=coverage.out ./...; \
		go tool cover -html=coverage.out -o coverage.html

build: deps update-certs
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build \
		-a -installsuffix cgo \
		-tags=jsoniter -o target/app .

update-certs:
		mkdir -p target && \
		curl -o target/ca-certificates.crt https://curl.haxx.se/ca/cacert.pem