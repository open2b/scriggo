module compare

replace scriggo => ../../

replace testpkg => ./testpkg

require (
	scriggo v0.0.0-00010101000000-000000000000
	testpkg v0.0.0-00010101000000-000000000000
)

go 1.13
