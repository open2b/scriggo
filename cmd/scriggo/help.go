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

will install an interpreter named "example" (or "example.exe") from the
commands in the file "github.com/organization/example/Scriggofile".

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
different name to the package use the instruction SET PACKAGE in the
Scriggofile:

SET PACKAGE example

For more about the Scriggofile specific format, see 'scriggo help Scriggofile'.

`

const helpScriggofile = `
A Scriggofile is a file with a specific format used by the scriggo command.
The scriggo command uses the instructions in a Scriggofile to build an
interpreter or a Go source file used in an application that embeds Scriggo.

A Scriggofile defines which packages an interpreted program or script can
import, what exported declarations in a package are accessible and so on.

It is a plain text file with encoding UTF-8 with an instruction per line. It
should be named 'Scriggofile' or with the extension '.Scriggofile' as in
'example.Scriggofile'.

The instructions are:

    IMPORT STANDARD LIBRARY 

        Makes the packages in the Go standard library (almost all) importable
        in a program or script executed by the interpreter.  

    IMPORT <package>

        Make the package with path <package> importable. 

    IMPORT <package> INCLUDING <A> <B> <C>

        As for 'IMPORT <package>' but only the exported names <A>, <B> and <C>
        are imported.

    IMPORT <package> EXCLUDING <A> <B> <C>

        As for 'IMPORT <package>' but the exported names <A>, <B> and <C> are
        not imported.  

    IMPORT <package> AS <as>

        As for 'IMPORT <package>' but the path with which it can be imported
        is named <as>. INCLUDING and EXCLUDING can be used as for the other
        forms of IMPORT at the end of the instruction.
    
    IMPORT <package> AS main

        Make the package with path <package> imported as the main package in a
        script or template. It is the same as writing 'import . "<package>"'
        in a Go program. INCLUDING and EXCLUDING can be used as for the other
        forms of IMPORT at the end of the instruction.

    IMPORT <package> AS main NOT CAPITALIZED

        As for 'IMPORT <package> AS main' but the exported names in the package
        will be imported not capitalized. For example a name 'FooFoo' declared
        in the package will be imported in the script or template as 'fooFoo'.

    TARGET PROGRAMS SCRIPTS TEMPLATES

        Indicates witch are the targets of the interpreter. It will be able to
        execute only the type of sources listed in the TARGET instruction. This
        instruction is only read by the 'build' and 'install' commands.

    SET VARIABLE <name> 

        Set the name of the variable to witch is assigned the value of type
        scriggo.PackageLoader with the packages to import. By default the name
        is 'packages'. This instruction is only read by the 'embed' command. 

    SET PACKAGE <name>

        Set the name of the package of the generated Go source file. By default
        the name of the package is 'main'. This instruction is read only by the
        command 'scriggo embed'.

    GOOS <linux> <windows>

        Specifies the operating systems that will be supported by the built
        interpreter. If the GOOS at the time the Scriggofile is parsed is not
        listed in the GOOS instruction, the 'build' and 'install' commands
        fail. If there is no GOOS instruction, all the operating systems are
        supported. 

        To view possible GOOS values run 'go tool dist list'.
`
