build: deps
	PATH=$(PATH):.bin protoc -I hellogrpc/ hellogrpc/hellogrpc.proto --go_out=plugins=grpc:hellogrpc

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
