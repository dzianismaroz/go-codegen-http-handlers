generate:
	go build handlers_gen/codegen.go && ./codegen  api.go api_handlers.go

test:
	go test -v
