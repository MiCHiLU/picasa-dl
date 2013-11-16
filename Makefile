PROJECT=picasa-dl
PLATFORMS=darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64
LOCALES=ja_JP

go: bin/$(PROJECT)

GO=$(wildcard src/$(PROJECT)/*.go)
MAPPING=$(wildcard locale/*.go.mapping)
$(MAPPING:.mapping=.pot): $(GO)
POT=$(MAPPING:.mapping=.pot) $(MAPPING:.mapping=.pot)
PO=$(wildcard locale/*/LC_MESSAGES/*.po)
MO=$(PO:.po=.mo)

.SUFFIXES: .po .mo
.po.mo:
	pybabel compile -d locale -D $(notdir $*)

.SUFFIXES: .mapping .pot
.mapping.pot:
	pybabel extract -k GetText -o $@ -F $< src/$(PROJECT)
	for locale in $(LOCALES); do\
		if [ -e $(dir $@)$$locale/LC_MESSAGES/$(notdir $(basename $@)).po ]; then\
			pybabel update -D $(notdir $*) -i $@ -d locale -l $$locale;\
		else\
			pybabel init   -D $(notdir $*) -i $@ -d locale -l $$locale;\
		fi;\
	done

bin/$(PROJECT): $(GO)
	go fmt $<
	go install -tags version_embedded -ldflags "-X main.version $$(git describe --always) -X main.buildAt '$$(LANG=en date -u +'%b %d %T %Y')'" $(PROJECT)

mo: $(MAPPING) $(POT) $(MO)

race: bin/$(PROJECT)
	go install -race $(PROJECT)

all: src/$(PROJECT)/*.go
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
	cd bin && zip -FS $(PROJECT)-unix.zip [dfl]*/$(PROJECT)
	cd bin && zip -FS $(PROJECT)-win.zip windows_*/$(PROJECT).exe

clean:
	rm -f ./bin/$(PROJECT)* ./bin/*/$(PROJECT)* ./bin/*.zip
