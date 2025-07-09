test:
	@go clean -testcache && go test ./internal/services -v
