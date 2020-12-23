
COMMONENVVAR=GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILDENVVAR=CGO_ENABLED=0

all: build


build: 
	$(COMMONENVVAR) $(BUILDENVVAR) go build -o ./bin/load-watcher main.go

local-image:
	docker build -t load-watcher .

clean:
	rm -rf ./bin
