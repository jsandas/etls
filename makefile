GOLANG_VERSION=$(shell cat go.mod | egrep "^go\s" | cut -d ' ' -f 2)

unit:
	go test -count=1 ./... -coverprofile=coverage.out -covermode=atomic

unit_docker:
	docker run -v ${PWD}:/go/src/tlstools -w /go/src/tlstools --rm golang:${GOLANG_VERSION} \
	bash -c "go test -count=1 ./... -coverprofile=coverage.out -covermode=atomic"
