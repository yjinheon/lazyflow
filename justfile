binary := "lazyflow"

default: build run

# build
build:
    go build -o {{binary}} ./cmd/lazyflow/

# run
run:
    ./{{binary}}

# build and run
dev: build
    ./{{binary}}

# test
test:
    go test ./...

# linting
lint:
    go vet ./...

# dependencies
tidy:
    go mod tidy

# rm binary
clean:
    rm -f {{binary}}
