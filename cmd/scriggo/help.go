// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const helpScriggo = `
Scriggo is an embeddable Go interpreter. The scriggo command is a tool that
can be used to build and install stand alone interpreters and to generate Go
source files useful to embed Scriggo in an application.

It also provides a web server that serves a template rooted at the current
directory, useful to learn Scriggo templates. See 'scriggo help serve'.

The scriggo tool is not required to embed Scriggo in an application but it is
useful to generate the code for a package loader used by the Scriggo Build
functions to load the packages that can be imported during the execution of a
program.

For more about the use of the scriggo command to embed Scriggo in an
application, see 'scriggo help embed'.

The scriggo command is also able to build and install stand alone interpreters
without having to write any line of Go. 

For more about to build interpreters, see 'scriggo help build' and
'scriggo help install'.

The commands are:

    embed       make a Go file with the source of a package loader useful when
                embedding Scriggo in an application

    build       build an interpreter starting from a Scriggofile     

    install     build and install an interpreter in the GOBIN directory

    serve       runs a web server and serves the template rooted at the current
                directory

    version     print Scriggo and scriggo version

    stdlib      print the packages imported by the instruction
                'IMPORT STANDARD LIBRARY' in the Scriggofile

Use 'scriggo help <command>' for more information about a command.

Additional help topics:

    Scriggofile     syntax of the Scriggofile
    
    limitations     limitations of the Scriggo compiler/runtime.
    
`

const helpBuild = `
usage: scriggo build [-f Scriggofile] [-w] [-v] [-x] [-work] [-o output] [module]

Build compiles an interpreter for Scriggo programs from a Scriggofile in a module.

Executables are created in the current directory. To install the executables in
the directory GOBIN, see the command: scriggo install.

If an argument is given, it can be a module path or a directory path.

If the argument is a module path, the module is downloaded from its repository
and the build command looks for a Scriggofile named 'Scriggofile' in its root.
A module argument can have a version as in 'foo.boo@v2.1.0'. If no version is
given the latest version of the module is downloaded.

If the argument is a directory path, it must be rooted or must begin with
a . or .. element and the directory must be the root of a module. The build
command looks for a Scriggofile named 'Scriggofile' in that directory.

If no argument is given, the action applies to the current directory.

The name of the executable is the last element of the module's path or
directory path. For example if the module's path is 'boo/foo' the name of the
executable will be 'foo' or 'foo.exe'.

The interpreter is build from the instructions in the Scriggofile. For example:

    scriggo build github.com/example/foo

will build an interpreter named 'foo' or 'foo.exe' from the instructions in
the file at 'github.com/example/foo/Scriggofile'.

In this other example:

	scriggo build ./boo

the command will build an interpreter named 'boo' or 'boo.exe' from the
instructions in the Scriggofile './boo/Scriggofile'.

For more about the Scriggofile specific format, see 'scriggo help Scriggofile'.

The -f flag forces build to read the given Scriggofile instead of the
Scriggofile of the module. For example:

    scriggo build -f boo.Scriggofile github.com/example/foo

will build an interpreter named 'foo' or 'foo.exe' from the instructions in
the file at 'boo.Scriggofile'.

The -w flag omits the DWARF symbol table.

The -v flag prints the imported packages as defined in the Scriggofile.

The -x flag prints the executed commands.

The -work flag prints the name of a temporary directory containing a work
module used to build the interpreter. The directory will not be deleted
after the build.

The -o flag forces build to write the resulting executable to the named output
file, instead in the current directory.

See also: scriggo install and scriggo embed.
`

const helpInstall = `
usage: scriggo install [-f Scriggofile] [-w] [-v] [-x] [-work] [module]

Install compiles and installs an interpreter for Scriggo programs
from a Scriggofile in a module.

Executables are installed in the directory GOBIN as for the go install
command.

For more about the GOBIN directory, see 'go help install'.

With the exception of the flag -o, install has the same parameters as build.
For more about the parameters, see 'scriggo help build'.

See also: scriggo build and scriggo embed.
`

const helpEmbed = `
usage: scriggo embed [-f Scriggofile] [-v] [-x] [-o output] [module]

Embed makes a Go source file from a Scriggofile in a module, containing the
exported declarations of the packages imported in the Scriggofile. The
generated file is useful when embedding Scriggo in an application.

Embed prints the generated source to the standard output. Use the flag -o
to redirect the source to a named output file.

If an argument is given, it must be a local rooted path or must begin with
a . or .. element and it must be a module root directory. embed looks for
a Scriggofile named 'Scriggofile' in that directory.

If no argument is given, the action applies to the current directory.

The -f flag forces embed to read the given Scriggofile instead of the
Scriggofile of the module.

The declarations in the generated Go file have type scriggo.PackageLoader and
they are assigned to a variable named 'packages'. The variable can be used as
an argument to the Build functions in the scriggo package.

To give a different name to the variable use the instruction SET VARIABLE in
the Scriggofile:

    SET VARIABLE foo

The package name in the generated Go file is by default 'main', to give
a different name to the package use the instruction SET PACKAGE in the
Scriggofile:

    SET PACKAGE boo

The -v flag prints the imported packages as defined in the Scriggofile.

The -x flag prints the executed commands.

The -o flag writes the generated Go file to the named output file, instead to
the standard output.

For more about the Scriggofile specific format, see 'scriggo help Scriggofile'.

`

const helpServe = `
usage: scriggo serve [-S] [--metrics]

Serve runs a web server and serves the template rooted at the current
directory. It is useful to learn Scriggo templates.

It renders HTML and Markdown files based on file extension.

For example:

    http://localhost:8080/article.html

renders the template file named 'article.html' as HTML and

    http://localhost:8080/blog.md

renders the template file named 'blog.md' as Markdown. Markdown is converted to
HTML with the Goldmark parser with the default options.

Templates are automatically rebuilt when a file changes.

The -S flag prints the assembly code of the served template.

The --metrics flags prints metrics about execution time.
`

const helpScriggofile = `
A Scriggofile is a file with a specific format used by the scriggo command.
The scriggo command uses the instructions in a Scriggofile to build an
interpreter or a Go source file used in an application that embeds Scriggo.

A Scriggofile defines which packages an interpreted program can import,
what exported declarations in a package are accessible and so on.

The format of the Scriggofile is:

    # A comment
    INSTRUCTION arguments

A line starting with '#' is a comment, and the instructions are case
insensitive but for convention are written in uppercase (the syntax recalls
that used by Dockerfile). 

A Scriggofile must be encoded as UTF-8 and it should be named 'Scriggofile'
or with the extension '.Scriggofile' as for 'example.Scriggofile'.

The instructions are:

    IMPORT STANDARD LIBRARY 

        Makes the packages in the Go standard library (almost all) importable
        in a program executed by the interpreter.

        To view all packages imported run 'scriggo stdlib'.

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
        forms of IMPORT at the end of the instruction. Is not possible to use
        a path <as> that would conflict with a Go standard library package path,
        even if this latter is not imported in the Scriggofile.
    
    IMPORT <package> AS main

        Make the package with path <package> imported as the main package.
        It is the same as writing 'import . "<package>"' in a Go program.
        INCLUDING and EXCLUDING can be used as for the other forms of IMPORT at
        the end of the instruction.

    IMPORT <package> AS main NOT CAPITALIZED

        As for 'IMPORT <package> AS main' but the exported names in the package
        will be imported not capitalized. For example a name 'FooFoo' declared
        in the package will be imported in the script as 'fooFoo'.

    SET VARIABLE <name> 

        Set the name of the variable to witch is assigned the value of type
        scriggo.PackageLoader with the packages to import. By default the name
        is 'packages'. This instruction is only read by the 'embed' command. 

    SET PACKAGE <name>

        Set the name of the package of the generated Go source file. By default
        the name of the package is 'main'. This instruction is read only by the
        command 'scriggo embed'.

    GOOS linux windows

        Specifies the operating systems that will be supported by the built
        interpreter. If the GOOS at the time the Scriggofile is parsed is not
        listed in the GOOS instruction, the 'build' and 'install' commands
        fail. If there is no GOOS instruction, all the operating systems are
        supported. 

        To view possible GOOS values run 'go tool dist list'.
`

const helpLimitations = `

Limitations

    These limitations are features that Scriggo currently lacks but that are
    under development. To check the state of a limitation please refer to the
    Github issue linked in the list below.

    * methods declarations (issue #458)
    * interface types definition (issue #218)
    * assigning return values from a deferred closure (issue #278)
    * assigning to non-variables in 'for range' statements (issue #182)
    * importing the "unsafe" package from Scriggo (issue #288)
    * importing the "runtime" package from Scriggo (issue #524)
    * labeled continue and break statements (issue #83)
    * struct type declarations with implicit fields (issue #367)
    * struct type declarations with tags (issue #61)
    * some kinds of pointer shorthands (issue #383)
    * compilation of non-main packages without importing them (issue #521)

    For a comprehensive list of not-yet-implemented features
    see https://github.com/open2b/scriggo/labels/missing-feature.

Limitations due to mantain the interoperability with Go official compiler 'gc'

    * types defined in Scriggo are not correctly seen by the 'reflect' package.
      This manifests itself, for example, when calling the function
      'fmt.Printf("%T", v)' where 'v' is a value with a Scriggo defined type.
      The user expects to see the name of the type but 'fmt' (which internally
      relies on the package 'reflect') prints the name of the type that wrapped
      the value in 'v' before passing it to gc.

    * not exported fields of struct types defined in Scriggo are still
      accessible from the outside.
      This is caused by the function 'reflect.StructOf' that requires that all
      fields are exported before creating the type.  By the way, such fields
      (that should be not exported) are exported with a particular prefix to
      avoid accidental accessing.

    * cannot define functions without a body (TODO)

    * Go packages can be imported only if they have been precompiled into the
      Scriggo interpreter/execution environment.
      Also see the commands 'scriggo embed' and 'scriggo build'.

    * the debugger 'delve' is not compatible with Scriggo; currently there are
      no debuggers available for Scriggo.

Arbitrary limitations

    These limitations have been arbitrarily added to Scriggo to enhance
    performances:

    * 127 registers of a given type (integer, floating point, string or
        general) per function
        * 256 function literal declarations plus unique functions calls per
        function
    * 256 types available per function
    * 256 unique predefined functions per function
    * 256 integer values per function
    * 256 string values per function
    * 256 floating-point values per function
    * 256 general values per function

`
