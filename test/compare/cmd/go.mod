module github.com/open2b/scriggo/cmd

replace scriggo => ../../../

replace testpkg => ../testpkg

go 1.13

require (
	scriggo v0.0.0-00010101000000-000000000000
	testpkg v0.0.0-00010101000000-000000000000
)
