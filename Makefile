# For people who don't want to use go's standard settings

.PHONY: lit lit-af glit test tests

GOBIN = $(shell pwd)
GO ?= latest
WEBUI_REPO = "https://github.com/josephtchung/webui"
all: lit test

goget:
	build/env.sh go get ./...

lit: goget
	build/env.sh go build
	@echo "Done building."
	@echo "Run \"$(GOBIN)/lit\" to launch lit."

lit-af: goget
	cd cmd/lit-af ; go build
	@echo "Run \"$(GOBIN)/cmd/lit-af/lit-af\" to launch lit-af."

glit: goget
	cd cmd/glit ; go build
	@echo "Run \"$(GOBIN)/cmd/glit/glit\" to launch glit."

webui:
	git clone $(WEBUI_REPO); cd webui ; npm install
	@echo "Run npm start from $(GOBIN)/webui and navigate to localhost:3000"

test tests: lit
	build/env.sh go test -v ./...
ifeq ($(with-python), true)
	python3 test/test_basic.py -c reg --dumplogs
	python3 test/test_break.py -c reg --
endif
