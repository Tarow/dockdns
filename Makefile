BINARY_NAME        := dockdns
IMAGE_NAME         := alex4108/$(BINARY_NAME)
TAGS               := latest
TEMPL_BIN 		   := ./bin/templ


all: clean install gen tidy build

run: gen
	go run main.go

build: gen
	go build -o bin/$(BINARY_NAME) main.go

$(TEMPL_BIN):
	GOBIN=$(PWD)/bin go install github.com/a-h/templ/cmd/templ@latest

gen: $(TEMPL_BIN)
	$(TEMPL_BIN) generate

clean:
	rm -f bin/*

install:
	go mod download

lint:
	@golangci-lint --version
	golangci-lint run

tidy:
	go mod tidy

docker-build: all
	docker build -f docker/Dockerfile . --tag $(IMAGE_NAME):$(firstword $(TAGS))
	$(foreach tag,$(filter-out $(firstword $(TAGS)),$(TAGS)),\
		docker tag $(IMAGE_NAME):$(firstword $(TAGS)) $(IMAGE_NAME):$(tag); \
	)

docker-push: docker-build
	$(foreach tag, $(TAGS),\
		docker push $(IMAGE_NAME):$(tag); \
	)

e2e-test: build
	bash test/e2e/run.sh