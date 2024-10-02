export GO111MODULE=on
export TF_LOG=DEBUG
SRC=$(shell find . -name '*.go')
SONARQUBE_IMAGE?=sonarqube:lts-developer
SONARQUBE_START_SLEEP?=60

.PHONY: all vet build test

all: fmt vet build

build:
	go build -a -tags netgo -o terraform-provider-sonarqube

fmt:
	go fmt ./...

vet:
	go vet ./...

testacc:
	docker run --name sonarqube1 -d -p 9001:9000 ${SONARQUBE_IMAGE}
	# timeout 300 bash -c 'while [[ "curl -s -o /dev/null -w ''%{http_code}'' localhost:9001/api/system/info" != "401" ]]; do echo "waiting for sonarqube to start"; sleep 15; done'
	sleep ${SONARQUBE_START_SLEEP}
	-TF_ACC=1 SONAR_HOST=http://localhost:9001 SONAR_USER=admin SONAR_PASS=admin go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	docker stop sonarqube1
	docker rm sonarqube1