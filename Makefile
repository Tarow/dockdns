BINARY_NAME        := dockdns
IMAGE_NAME         := ghcr.io/tarow/$(BINARY_NAME)
TAGS               := latest


all: clean install gen tidy build

run: gen
	go run main.go

build: gen
	go build -o bin/$(BINARY_NAME) main.go

gen:
	templ generate

clean:
	rm -f bin/*

install:
	go mod download

lint:
	@golangci-lint --version
	golangci-lint run

tidy:
	go mod tidy


docker-build:
	docker build -f docker/Dockerfile . --tag $(IMAGE_NAME):$(firstword $(TAGS))
	$(foreach tag,$(filter-out $(firstword $(TAGS)),$(TAGS)),\
		docker tag $(IMAGE_NAME):$(firstword $(TAGS)) $(IMAGE_NAME):$(tag); \
	)

docker-push:
	$(foreach tag, $(TAGS),\
		docker push $(IMAGE_NAME):$(tag); \
	)


