run-server: gen
	go run ./server

run-client: gen
	go run ./client

gen: deps
	PATH=$(PATH):.bin protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	remote/remote.proto

deps: protoc protoc-gen-go protoc-gen-go-gprc

protoc: .bin/protoc

.bin/protoc:
	@{ \
	ARCHIVE=protoc-3.17.0-linux-x86_64.zip ;\
	curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.17.0/$$ARCHIVE ;\
	mkdir -p .bin ;\
	unzip -j -o $$ARCHIVE -d .bin bin/protoc ;\
	rm -f $$ARCHIVE ;\
	}

protoc-gen-go: .bin/protoc-gen-go

.bin/protoc-gen-go:
	GOBIN=$(abspath .bin) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26

protoc-gen-go-gprc: .bin/protoc-gen-go-grpc

.bin/protoc-gen-go-grpc:
	GOBIN=$(abspath .bin) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
