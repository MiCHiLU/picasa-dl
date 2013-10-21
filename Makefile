bin/picasa-dl: src/picasa-dl/*.go
	go fmt $<
	go install picasa-dl
