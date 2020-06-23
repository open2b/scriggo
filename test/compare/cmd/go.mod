module github.com/open2b/scriggo/cmd

replace github.com/open2b/scriggo => ../../../

replace testpkg => ../testpkg

go 1.14

require (
	github.com/open2b/scriggo v0.0.0
	testpkg v0.0.0-00010101000000-000000000000
)
