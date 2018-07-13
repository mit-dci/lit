GOBIN = $(shell pwd)
GO ?= latest

GO_BUILD_EX_ARGS ?=

all: lit lit-af test

.PHONY: lit lit-af glit test tests webui

goget:
	build/env.sh go get -v .
	build/env.sh go get -v ./cmd/lit-af

lit: goget
	build/env.sh go build -v ${GO_BUILD_EX_ARGS} .
	@echo "Done building."
	@echo "Run \"$(GOBIN)/lit\" to launch lit."

lit-af: goget
	build/env.sh go build -v ${GO_BUILD_EX_ARGS} ./cmd/lit-af
	@echo "Run \"$(GOBIN)/lit-af\" to launch lit-af."

glit:
	build/env.sh go get -v ./cmd/glit
	build/env.sh bash -c '(cd cmd/glit && go build -v)'
	@echo "Run \"$(GOBIN)/glit\" to launch glit."

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
	rm -rf webui/
	rm -f cmd/lit-af/lit-af
	rm -f cmd/glit/glit

test: lit
	build/env.sh ./gotests.sh
ifeq ($(with-python), true)
	python3 test/test_basic.py -c reg --dumplogs
	python3 test/test_break.py -c reg --dumplogs
endif
