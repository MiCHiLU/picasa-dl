PROJECT=picasa-dl
PLATFORMS=darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64

bin/$(PROJECT): src/$(PROJECT)/*.go
	go fmt $<
	go install -tags version_embedded -ldflags "-X main.version $$(git describe --always) -X main.buildAt '$$(LANG=en date -u)'" $(PROJECT)

race: bin/$(PROJECT)
	go install -race $(PROJECT)

all: src/$(PROJECT)/*.go
	make clean
	@failures="";\
	for platform in $(PLATFORMS); do\
	  echo building for $$platform;\
	  GOOS=$${platform%/*} GOARCH=$${platform#*/} go install -tags version_embedded -ldflags "-X main.version $$(git describe --always) -X main.buildAt '$$(LANG=en date -u)'" $(PROJECT) || failures="$$failures $$platform";\
	done;\
	if [ "$$failures" != "" ]; then\
	  echo "*** FAILED on $$failures ***";\
	  exit 1;\
	fi

clean:
	rm -f ./bin/$(PROJECT)* ./bin/*/$(PROJECT)*
