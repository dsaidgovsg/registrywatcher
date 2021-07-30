# Setup name variables for the package/tool
NAME := registrywatcher
PKG := github.com/dsaidgovsg/$(NAME)

.PHONY: snakeoil
snakeoil: ## Update snakeoil certs for testing.
	go run $(GOROOT)/src/crypto/tls/generate_cert.go --host localhost,127.0.0.1 --ca
	mkdir -p testutils/snakeoil
	mv cert.pem $(CURDIR)/testutils/snakeoil/cert.pem
	mv key.pem $(CURDIR)/testutils/snakeoil/key.pem
