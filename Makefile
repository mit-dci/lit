# For people who don't want to use go's standard settings

.PHONY: lit lit-af glit test tests webui

GOBIN = $(shell pwd)
GO ?= latest
all: lit test

goget:
	build/env.sh go get ./...

lit: goget
	build/env.sh go build
	@echo "Done building."
	@echo "Run \"$(GOBIN)/lit\" to launch lit."

lit-af: goget
	cd cmd/lit-af ; go get ./... ; go build
	@echo "Run \"$(GOBIN)/cmd/lit-af/lit-af\" to launch lit-af."

webui:
	cd webui ; rm -rf node_modules/ ; npm install ; npm run build ; cd ..
	@echo "Launch app from ./webui/dist/<your_dist>/litwebui"

test tests: lit
	build/env.sh go test -v ./...
ifeq ($(with-python), true)
	python3 test/test_basic.py -c reg --dumplogs
	python3 test/test_break.py -c reg --dumplogs
endif
