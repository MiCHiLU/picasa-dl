bin/picasa-dl: src/picasa-dl/*.go
	go fmt $<
	go install picasa-dl

race: bin/picasa-dl
	go install -race picasa-dl
