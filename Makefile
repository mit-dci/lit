# For people who don't want to use go's standard settings

.PHONY: lit lit-af glit test clean

GOBIN = $(shell pwd)
GO ?= latest
WEBUI_REPO = "https://github.com/josephtchung/webui"

all: lit lit-af

lit:
	build/env.sh go get -v .
	build/env.sh go build -v
	@echo "Done building."
	@echo "Run \"$(GOBIN)/lit\" to launch lit."

lit-af:
	build/env.sh go get -v ./cmd/lit-af
	build/env.sh bash -c '(cd cmd/lit-af && go build -v)'
	@echo "Run \"$(GOBIN)/cmd/lit-af/lit-af\" to launch lit-af."

glit:
	build/env.sh go get -v ./cmd/glit
	build/env.sh bash -c '(cd cmd/glit && go build -v)'
	@echo "Run \"$(GOBIN)/cmd/glit/glit\" to launch glit."

webui:
	git clone $(WEBUI_REPO) webui
	cd webui
	npm install
	@echo "Run npm start from $(GOBIN)/webui and navigate to localhost:3000"

clean:
	rm -rf build/_workspace/
	rm -rf webui/
	rm -f lit
	rm -f cmd/lit-af/lit-af
	rm -f cmd/glit/glit

test: lit
	build/env.sh go test -v ./...
ifeq ($(with-python), true)
	python3 test/test_basic.py -c reg --dumplogs
	python3 test/test_break.py -c reg --dumplogs
endif
