GOLANG_VERSION=$(shell cat go.mod | egrep "^go\s" | cut -d ' ' -f 2)

unit:
	go test -count=1 ./... -coverprofile=coverage.out -covermode=atomic

unit_docker:
	docker pull golang:${GOLANG_VERSION}-bullseye
	docker run -v ${PWD}:/go/src/etl -w /go/src/etl --rm golang:${GOLANG_VERSION}-bullseye \
	bash -c "go test -count=1 ./... -coverprofile=coverage.out -covermode=atomic"
