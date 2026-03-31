BINARY=terraform-provider-procurator

build:
	go build -o $(BINARY) .

build-grpc:
	go build -tags grpcapi -o $(BINARY) .

fmt:
	gofmt -w .
