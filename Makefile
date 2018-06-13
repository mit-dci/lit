# For people who're too lazy to use go's standard settings

.PHONY: lit

GOBIN = $(shell pwd)/build/bin
GO ?= latest

lit:
	build/env.sh go get . go build
	@echo "Done building."
	@echo "Run \"$(GOBIN)/lit\" to launch lit."
