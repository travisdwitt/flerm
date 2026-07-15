BINARY := flerm

.PHONY: build install run test fmt clean

build:
	go build -o $(BINARY) ./cmd/flerm

install:
	go install ./cmd/flerm

run:
	go run ./cmd/flerm

test:
	go test ./...

fmt:
	gofmt -w cmd internal

clean:
	rm -f $(BINARY)
