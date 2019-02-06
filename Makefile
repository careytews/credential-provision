
VERSION=$(shell git describe | sed 's/^v//')

CONTAINER=gcr.io/trust-networks/credential-mgmt:${VERSION}

CERT_TOOLS = create-cert create-cert-request \
	create-ca-cert create-crl create-key \
	find-cert

GOFILES = decode encode-secret encode-file upload-to-storage encode-key \
	setup-ckms generate-key destroy-ckms download-from-storage \
	credential-provision delete-from-storage upload-crl-to-storage update-index-file

CORE = credential-common.go

all: ${GOFILES} ${GODEPS} container

GODEPS= go/.oauth2 go/.pbkdf2 go/.cloudkms go/.goflags \
	go/.pubsub go/.uuid go/.cert-tools

%: %.go ${GODEPS} 
	GOPATH=$$(pwd)/go go build $< ${CORE}

go:
	mkdir go

go/.cert-tools: go/.uuid go/.goflags
	GOPATH=$$(pwd)/go go get -u github.com/trustnetworks/certificate-tools
	GOPATH=$$(pwd)/go GOBIN=$$(pwd) go install github.com/trustnetworks/certificate-tools/...
	touch $@

go/.uuid:
	GOPATH=$$(pwd)/go go get github.com/google/uuid
	touch $@

go/.goflags:
	GOPATH=$$(pwd)/go go get github.com/jessevdk/go-flags
	touch $@

go/.oauth2:
	GOPATH=$$(pwd)/go go get golang.org/x/oauth2
	GOPATH=$$(pwd)/go go get golang.org/x/oauth2/google
	touch $@

go/.pbkdf2:
	GOPATH=$$(pwd)/go go get golang.org/x/crypto/pbkdf2
	touch $@

go/.pubsub:
	GOPATH=$$(pwd)/go go get google.golang.org/api/pubsub/v1
	touch $@

go/.cloudkms:
	GOPATH=$$(pwd)/go go get google.golang.org/api/cloudkms/v1
	touch $@

container: ${GODEPS} ${GOFILES}
	docker build -t ${CONTAINER} \
	  -f Dockerfile .

run: container
	docker run --rm -i -t \
		-v $$(pwd)/ca:/ca -v $$(pwd)/ca_cert:/ca_cert \
		-v $$(pwd)/key:/key \
		-e CA=/ca -e CA_CERT=/ca_cert \
		gcr.io/trust-networks/credential-mgmt:${VERSION} bash

push: container
	gcloud docker -- push ${CONTAINER}

clean:
	rm -rf ${GOFILES} ${CERT_TOOLS}
	rm -rf go
	rm -rf cert-tools

