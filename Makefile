PROJECT=picasa-dl
PLATFORMS=darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64

bin/$(PROJECT): src/$(PROJECT)/*.go
	go fmt $<
	go install $(PROJECT)

race: bin/$(PROJECT)
	go install -race $(PROJECT)

all: src/$(PROJECT)/*.go
	@failures="";\
	for platform in $(PLATFORMS); do\
	  echo GOOS=$${platform%/*} GOARCH=$${platform#*/} go install $(PROJECT);\
	  GOOS=$${platform%/*} GOARCH=$${platform#*/} go install $(PROJECT) || failures="$$failures $$platform";\
	done;\
	if [ "$$failures" != "" ]; then\
	  echo "*** FAILED on $$failures ***";\
	  exit 1;\
	fi
