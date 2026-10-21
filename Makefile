.PHONY: lint test e2e

lint:
	golangci-lint run --build-tags e2e ./...

test:
	go test -race ./...

# The e2e tests need Docker. Set E2E_PHP_IMAGE to use a different php-fpm image.
# The tests build the wrapper at run time, so the test cache cannot see a change of the code.
e2e:
	go test -race -count=1 -tags e2e ./e2e/...
