// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const helpDescriptor = `
A Scriggo descriptor file consits of a valid Go package source code containing one Scriggo file comment and one or more imports, which may in turn have a Scriggo import comment

An example Scriggo descriptor is:

	//scriggo: interpreters:"script"
	
	package x
	
	import (
		_ "fmt"
		_ "math" //scriggo: main uncapitalize
	)

This Scriggo descriptor describes a Scriggo interpreter provides package "fmt" (available through an import statement) and package "math" as "builtin", with all names "uncapitalized".

Each import statement should have a name _, which prevents tools like goimports from removing import.

Options available in the Scriggo file comment are:

	interpreters[:targets]  describe an interpreter for targets. Valid targets are "template, "script" and "program". If not targets are specified, it's assumed by default all available interpreters.
	embedded                describe an embedded packages declaration
	output                  select output file/directory
	goos:GOOSs              force GOOS to the specified value. More than one value can be provided

Options available as Scriggo import comments are:

	main                    import as package main. Only available in scripts an templates
	uncapitalize            declarations imported as main are "uncapitalized"
	path                    force an alternative path for import
	export:names            only export names
	noexport:names          export everything excluding names

Example import comments

Default. Makes "fmt" available in Scriggo as would be available in Go:

	import _ "fmt" //scriggo:

Import all declarations from "fmt" in package main, making them accessible without a selector:

	import _ "fmt" //scriggo: main

Import all declarations from "fmt" in package main with uncapitalized names, making them accessible without a selector:

	import _ "fmt" //scriggo: main uncapitalize

Import all declarations from "fmt" excluding "Print" and Println":

	import _ "fmt" //scriggo: noexport:"Print,Println"
`
