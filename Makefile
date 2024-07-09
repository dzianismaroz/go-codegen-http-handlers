generate:
	go build -o ./codegen handlers_gen/*.go && ./codegen  api.go api_handlers.go

test:
	go test -v
