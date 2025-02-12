GOBIN ?= $$(go env GOPATH)/bin

test:
	go test ./... -v

install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

check-coverage: install-go-test-coverage
	go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=${PWD}/.testcoverage.yml
