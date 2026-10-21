.PHONY: lint test e2e

lint:
	golangci-lint run --build-tags e2e ./...

test:
	go test -race ./...

# The e2e tests need Docker. Set E2E_PHP_IMAGE to use a different php-fpm image.
e2e:
	go test -race -tags e2e ./e2e/...
