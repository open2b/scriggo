// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const helpBuild = `
usage: scriggo build [-v] [-work] package
       scriggo build [-v] [-work] file

Build compiles an interpreter for Scriggo programs, scripts and templates from
a package or single file.

Executables are created in the current directory. To install the executables in
the directory GOBIN, see the command: scriggo install.

If a package has a file named "Scriggofile" in its directory, an interpreter
is build and installed from the instructions in this file according to a
specific format. For example:

scriggo build github.com/organization/example

will an interpreter named "example" (or "example.exe") from the commands in the
file "github.com/organization/example/Scriggofile".

For more about the Scriggofile specific format, see 'scriggo help Scriggofile'.

If a file is given, instead of a package, the file must contains the commands
to build the interpreter. The name of the executable is the same of the file
without the file extension. For example if the file is "example.Scriggofile"
the executable will be named "example" (or "example.exe").

The -v flag prints the imported packages as defined in the Scriggofile.

The -work flag prints the name of a temporary directory containing a work
package used to build the interpreter. The directory will not be deleted
after the build.

See also: scriggo install and scriggo embed.
`

const helpInstall = `
usage: scriggo install [-v] package
       scriggo install [-v] file

Install compiles and installs an interpreter for Scriggo programs, scripts
and templates from a package or single file.

Executables are installed in the directory GOBIN as for the go install
command.

For more about the GOBIN directory, see 'go help install'.

If a package has a file named "Scriggofile" in its directory, an interpreter
is build and installed from the instructions in this file according to a
specific format. For example:

scriggo install github.com/organization/example

will install an interpreter named "example" (or "example.exe") from the commands
in the file "github.com/organization/example/Scriggofile".

For more about the Scriggofile specific format, see 'scriggo help Scriggofile'.

If a file is given, instead of a package, the file must contains the commands
to build the interpreter. The name of the executable is the same of the file
without the file extension. For example if the file is "example.Scriggofile"
the executable will be named "example" (or "example.exe").

The -v flag prints the imported packages as defined in the Scriggofile.

See also: scriggo build and scriggo embed.
`

const helpEmbed = `
usage: scriggo embed [-o output] file

Embed makes a Go source file from a Scriggofile containing the exported
declarations of the packages imported in the Scriggofile. The generated
file is useful when embedding Scriggo in an application.

Embed prints the generated source to the standard output. Use the flag -o
to redirect the source to a file.

The declarations in the Go source have type scriggo.PackageLoader and are
assigned to a variable names 'packages'. The variable can be used as argument
to the Load functions in the scriggo package.

To give to the variable a different name use the instruction SET VARIABLE in
the Scriggofile:

SET VARIABLE decl

The name of the package in the Go source is by default 'main', to give a
different name to the package use the instruction SET PACKAGE in the Scriggofile:

SET PACKAGE example

For more about the Scriggofile specific format, see 'scriggo help Scriggofile'.

`

const helpScriggofile = `
A Scriggo descriptor file consits of a valid Go package source code containing one
Scriggo file comment and one or more imports, which may in turn have a Scriggo import comment

An example Scriggo descriptor is:

	//scriggo: interpreters:"script"
	
	package x
	
	import (
		_ "fmt"
		_ "math" //scriggo: main uncapitalize
	)

This Scriggo descriptor describes a Scriggo interpreter provides package "fmt"
(available through an import statement) and package "math" as "builtin", with
all names "uncapitalized".

Each import statement should have a name _, which prevents tools like goimports from removing import.

Options available in the Scriggo file comment are:

	interpreters:targets  describe an interpreter for targets. Valid targets are "template, "script" and "program"
	interpreter           install all kinds of interpreters
	embedded              describe an embedded packages declaration
	output                select output file/directory
	goos:GOOSs            force GOOS to the specified value. More than one value can be provided

Options available as Scriggo import comments are:

	main                    import as package main. Only available in scripts an templates
	uncapitalize            declarations imported as main are "uncapitalized"
	path                    change Scrigo import path
	export:names            only export names
	noexport:names          export everything excluding names

Example import comments

Default. Makes "fmt" available in Scriggo as would be available in Go:

	import _ "fmt" //scriggo:

Import all declarations from "fmt" in package main, making them accessible
without a selector:

	import _ "fmt" //scriggo: main

Import all declarations from "fmt" in package main with uncapitalized names,
making them accessible without a selector:

	import _ "fmt" //scriggo: main uncapitalize

Import all declarations from "fmt" excluding "Print" and Println":

	import _ "fmt" //scriggo: noexport:"Print,Println"
`
