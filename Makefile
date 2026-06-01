.PHONY: build run test clean install

BINARY=tractor-vin

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)

install: build
	cp $(BINARY) /usr/local/bin/
