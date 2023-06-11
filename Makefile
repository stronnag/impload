prefix ?= $(HOME)/.local
APP = impload
GOFILES = btaddr_linux.go impload.go msp.go util.go btaddr_other.go mission.go mission-read.go  outfmt.go enumerate_port.go

IMPTAG=$(shell git describe --tag 2>/dev/null||echo notag)
IMPDATE=$(shell date +%F)
IMPID=$(shell git rev-parse  --short HEAD 2>/dev/null||echo unknown)
LDFLAGS="-s -w -X \"main.GitCommit=$(IMPID) / $(IMPDATE)\" -X \"main.GitTag=$(IMPTAG)\""

all: $(APP)

$(APP):  $(APP).go $(GOFILES) go.sum
	CGO_ENABLED=0  go build -trimpath -ldflags $(LDFLAGS)

go.sum: go.mod
	go mod tidy

install: $(APP)
	install -d $(prefix)/bin
	install $(APP) $(prefix)/bin/

clean:
	rm -f $(APP)
