PROJECT=picasa-dl
PLATFORMS=darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64
LOCALES=ja

.PHONY: all clean test

go: bin/$(PROJECT)

VERSION=$(shell git describe --tag)
PROJECTDIR=src/$(PROJECT)
LOCALEDIR=$(PROJECTDIR)/locale
PY_PACKAGES_DIR=python
BABEL=./bin/pybabel

GO=$(wildcard $(PROJECTDIR)/*.go)
MAPPING=$(wildcard $(LOCALEDIR)/*.go.mapping)
POT=$(MAPPING:.mapping=.pot)
PO=$(wildcard $(LOCALEDIR)/*/LC_MESSAGES/*.po)
MO=$(PO:.po=.mo)
LOCALE=$(MO:.mo=.mogo)

.SUFFIXES: .mapping .pot
.mapping.pot:
	$(BABEL) extract -k GetText -o $@ -F $< $(PROJECTDIR)
	@for locale in $(LOCALES); do\
		subcommand=init;\
		if [ -e $(dir $@)$$locale/LC_MESSAGES/$(notdir $(basename $@)).po ]; then\
			subcommand=update;\
		fi;\
		cmd="$(BABEL) $$subcommand -D $(notdir $*) -i $@ -d $(LOCALEDIR) -l $$locale";\
		echo $$cmd;\
		$$cmd;\
	done

.SUFFIXES: .po .mo
.po.mo:
	$(BABEL) compile -d $(LOCALEDIR) -D $(notdir $*)

.SUFFIXES: .mo .mogo
.mo.mogo:
	./bin/go-bindata -func=Mo -nomemcopy -out=$@ -pkg=ja $<
	mkdir -p  $(LOCALEDIR)/ja
	cp $@ $(LOCALEDIR)/ja/ja.go

$(POT): $(GO)
$(PO): $(POT)
$(MO): $(POT)
$(LOCALE): $(MO)

bin/$(PROJECT): $(GO) $(LOCALE)
	go fmt $(PROJECT)
	go install -tags version_embedded -ldflags "-X main.version $$(git describe --always) -X main.buildAt '$$(LANG=en date -u +'%b %d %T %Y')'" $(PROJECT)
	go test -test.short $(PROJECT)

test:
	go test $(PROJECT)

race: bin/$(PROJECT)
	go install -race $(PROJECT)

all: $(GO) $(LOCALE) test
	make clean
	@failures="";\
	for platform in $(PLATFORMS); do\
	  echo building for $$platform;\
	  GOOS=$${platform%/*} GOARCH=$${platform#*/} go install -tags version_embedded -ldflags "-X main.version $$(git describe --always) -X main.buildAt '$$(LANG=en date -u +'%b %d %T %Y')'" $(PROJECT) || failures="$$failures $$platform";\
	done;\
	if [ "$$failures" != "" ]; then\
	  echo "*** FAILED on $$failures ***";\
	  exit 1;\
	fi
	cd bin && mkdir -p darwin_amd64 && cp -p $(PROJECT) darwin_amd64
	cd bin && zip -FS $(PROJECT)-$(VERSION)-unix.zip [dfl]*/$(PROJECT)
	cd bin && zip -FS $(PROJECT)-$(VERSION)-win.zip windows_*/$(PROJECT).exe

clean:
	rm -f $(POT) $(MO) $(LOCALE) ./bin/$(PROJECT)* ./bin/*/$(PROJECT)* ./bin/*.zip

setup:
	pip install --use-mirrors -t $(PY_PACKAGES_DIR) -r packages.txt
