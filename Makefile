PROJECT=picasa-dl
PLATFORMS=darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64

bin/$(PROJECT): src/$(PROJECT)/*.go
	go fmt $<
	go install $(PROJECT)

race: bin/$(PROJECT)
	go install -race $(PROJECT)

