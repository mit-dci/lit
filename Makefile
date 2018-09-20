GOBIN = $(shell pwd)
GO ?= latest

GO_BUILD_EX_ARGS ?=

all: lit lit-af test

.PHONY: lit lit-af test tests webui

goget:
	build/env.sh go get -v ./...

lit: goget
	build/env.sh go build ${GO_BUILD_EX_ARGS}
	@echo "Done building."
	@echo "Run \"$(GOBIN)/lit\" to launch lit."

lit-af: goget
	build/env.sh go build ${GO_BUILD_EX_ARGS} ./cmd/lit-af
	@echo "Run \"$(GOBIN)/lit-af\" to launch lit-af."

webui:
	cd webui ; rm -rf node_modules/ ; npm install ; npm run build ; cd ..
	@echo "Launch app from ./webui/dist/<your_dist>/litwebui"

package:
	build/releasebuild.sh clean
	build/releasebuild.sh

clean:
	build/env.sh clean
	build/releasebuild.sh clean
	go clean .
	go clean ./cmd/lit-af
	rm -rf build/_workspace/
	rm -f cmd/lit-af/lit-af

test tests: lit
	build/env.sh go test -v ./...
ifeq ($(with-python), true)
	cd test && mkdir -p _data && bash -c ./runtests.py
endif
