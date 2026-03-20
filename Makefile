.PHONY: build test vet proto clean

PROTO_DIR ?= ../protos
DESCRIPTOR := internal/grpcproxy/descriptors/sc.pb

build:
	go build ./...

test:
	go test ./... -count=1

vet:
	go vet ./...

# Compile all SC proto definitions into a single descriptor set.
# Requires: protoc, extracted .proto files in PROTO_DIR
proto:
	protoc \
		--descriptor_set_out=$(DESCRIPTOR) \
		--include_imports \
		--proto_path=$(PROTO_DIR) \
		$$(find $(PROTO_DIR) -name '*.proto')
	@echo "Compiled $$(wc -c < $(DESCRIPTOR)) bytes → $(DESCRIPTOR)"

clean:
	rm -f $(DESCRIPTOR)
	go clean ./...
