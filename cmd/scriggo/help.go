// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const helpScriggo = `
Scriggo is a template engine and Go interpreter. The scriggo command is a tool
that can be used to execute a template, initialize an interpreter, generate
the source files for a package importer and also provides a web server that
serves a template rooted at the current directory, useful to learn Scriggo
templates.

For more about executing a template, see 'scriggo help run'.

The commands are:

    run         run a template

    serve       run a web server and serve the template rooted at the current
                directory

    init        initialize an interpreter for Go programs

    import      generate the source for an importer used by Scriggo to import 
                a package when an 'import' statement is executed

    version     print the scriggo command version

    stdlib      print the packages imported by the instruction
                'IMPORT STANDARD LIBRARY' in the Scriggofile

Use 'scriggo help <command>' for more information about a command.

Additional help topics:

    Scriggofile     syntax of the Scriggofile

    limitations     limitations of the Scriggo compiler/runtime

`

const helpInit = `
usage: scriggo init [dir]

Init initializes an interpreter for Go programs creating the files in the
directory dir. If no argument is given, the files are created in the current
directory.

It creates in the directory:

* a go.mod file, if it does not already exist, with the directory name as
  module path

* a packages.go file with the native packages that can be imported with the
  import statement in the interpreted code

* a main.go file with the 'main' function

* a Scriggofile named 'Scriggofile' with the instructions to import the
packages of the standard library

then it calls the 'go mod tidy' command.

The interpreter can then be compiled with the 'go build' command or installed
with the 'go install' command.

For example:

    scriggo init ./example

will initialize an interpreter in the directory 'example'.

In the directory 'example', executing the command

    go install

an executable interpreter is installed in the 'bin' directory of the GOPATH with
name 'example' ('example.exe' on Windows).

To execute a Go source file called 'program.go' with the 'example' interpreter,
execute the command:

    example program.go

You can change the Scriggofile to import other packages and than use the
scriggo import command to rebuild the packages.go file with the package
importer.

For more about the Scriggofile specific format, see 'scriggo help Scriggofile'.

See also: scriggo import.
`

const helpImport = `
usage: scriggo import [-f Scriggofile] [-v] [-x] [-o output] [module]

Import generate the code for a package importer. An importer is used by Scriggo
to import a package when an 'import' statement is executed.

The code for the importer is generated from the instructions in a Scriggofile.
The Scriggofile should be in a Go module.

Import prints the generated source to the standard output. Use the flag -o
to redirect the source to a named output file.

If an argument is given, it must be a local rooted path or must begin with
a . or .. element and it must be a module root directory. import looks for
a Scriggofile named 'Scriggofile' in that directory.

If no argument is given, the action applies to the current directory.

The -f flag forces import to read the given Scriggofile instead of the
Scriggofile of the module.

The importer in the generated Go file have type native.Importer and it is
assigned to a variable named 'packages'. The variable can be used as an
argument to the Build and BuildTemplate functions in the scriggo package.

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

const helpRun = `
usage: scriggo run [-o output] [run flags] file

Run runs a template file and its extended, imported and rendered files.

For example:

    scriggo run article.html

runs the file 'article.html' as HTML and prints the result to the standard
output. Extended, imported and rendered file paths are relative to the
directory of the executed file.

The -o flag writes the result to the named output file or directory, instead to
the standard output.

Markdown is converted to HTML with the Goldmark parser with the options
html.WithUnsafe, parser.WithAutoHeadingID and extension.GFM.

The run flags are:

	-root dir
		set the root directory to dir instead of the file's directory.
	-const name=value
		run the template file with a global constant with the given name and
		value. name should be a Go identifier and value should be a string
		literal, a number literal, true or false. There can be multiple
		name=value pairs.
	-format format
		use the named file format: Text, HTML, Markdown, CSS, JS or JSON.
	-metrics
		print metrics about execution time.
	-S n
		print the assembly code of the executed file to the standard error.
		n determines the maximum length, in runes, of disassembled Text
		instructions:

		n > 0: at most n runes; leading and trailing white space are removed
		n == 0: no text
		n < 0: all text

Examples:

	scriggo run index.html

	scriggo run -const 'version=1.12 title="The ancient art of tea"' index.md

	scriggo run -root . docs/article.html

	scriggo run -format Markdown index

	scriggo run -o ./public ./sources/index.html

`

const helpServe = `
usage: scriggo serve [-S n] [--metrics] [--disable-livereload]

Serve runs a web server and serves the template rooted at the current
directory. It is useful to learn Scriggo templates.

It renders HTML and Markdown files based on file extension.

For example:

    http://localhost:8080/article

it renders the file 'article.html' as HTML if exists, otherwise renders the
file 'article.md' as Markdown.

Serving a URL terminating with a slash:

    http://localhost:8080/blog/

it renders 'blog/index.html' or 'blog/index.md'.

Markdown is converted to HTML with the Goldmark parser with the options
html.WithUnsafe, parser.WithAutoHeadingID and extension.GFM.

When a file is modified, the server automatically rebuilds templates, and the
browser reloads the page.

The -S flag prints the assembly code of the served file and n determines the
maximum length, in runes, of disassembled Text instructions

    n > 0: at most n runes; leading and trailing white space are removed
    n == 0: no text
    n < 0: all text

The --metrics flags prints metrics about execution time.

The --disable-livereload flag disables LiveReload, preventing automatic page
reloads in the browser.
`

const helpScriggofile = `
A Scriggofile is a file with a specific format used by the scriggo command.
The scriggo command uses the instructions in a Scriggofile to initialize an
interpreter or a Go source file used in an application that embeds Scriggo.

A Scriggofile defines which packages an interpreted program can import, what
exported declarations in a package are accessible and so on.

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
        in the package will be imported in a template file as 'fooFoo'.

    SET VARIABLE <name> 

        Set the name of the variable to witch is assigned the value of type
        scriggo.PackageImporter with the packages to import. By default the
        name is 'packages'. This instruction is only read by the 'import'
        command. 

    SET PACKAGE <name>

        Set the name of the package of the generated Go source file. By default
        the name of the package is 'main'. This instruction is read only by the
        command 'scriggo import'.

    GOOS linux windows

        Specifies the operating systems that will be supported by the built
        interpreter. If the GOOS at the time the Scriggofile is parsed is not
        listed in the GOOS instruction, the 'init' and 'import' commands
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
    * assigning to non-variables in 'for range' statements (issue #182)
    * importing the "unsafe" package from Scriggo (issue #288)
    * importing the "runtime" package from Scriggo (issue #524)
    * labeled continue and break statements (issue #83)
    * some kinds of pointer shorthands (issue #383)
    * compilation of non-main packages without importing them (issue #521)

    For a comprehensive list of not-yet-implemented features
    see https://github.com/open2b/scriggo/labels/missing-feature.

Limitations due to maintain the interoperability with Go official compiler 'gc'

    * types defined in Scriggo are not correctly seen by the 'reflect' package.
      This manifests itself, for example, when calling the function
      'fmt.Printf("%T", v)' where 'v' is a value with a Scriggo defined type.
      The user expects to see the name of the type but 'fmt' (which internally
      relies on the package 'reflect') prints the name of the type that wrapped
      the value in 'v' before passing it to gc.

    * unexported fields of struct types defined in Scriggo are still accessible
      from native packages with the reflect methods. This is caused by the
      reflect methods that does not allow, by design, to change the value of an
      unexported field, so they are created with an empty package path. By the
      way, such fields (that should be unexported) can not be changed without
      the reflect and have a particular prefix to avoid accidental accessing.

    * in a structure, types can be embedded but, apart from interfaces, if they
      have methods then they must be the first field of the struct. This is a
      limitation of the StructOf function of reflect.
      See Go issue #15924 (https://github.com/golang/go/issues/15924).

    * cannot define functions without a body (TODO)

    * a select supports a maximum of 65536 cases.

    * Native packages can be imported only if they have been precompiled into
      the Scriggo interpreter/execution environment.
      Also see the commands 'scriggo import' and 'scriggo init'.

    * types are not garbage collected.
      See Go issue #28783 (https://github.com/golang/go/issues/28783).

Arbitrary limitations

    These limitations have been arbitrarily added to Scriggo to enhance
    performances:

    * 127 registers of a given type (integer, floating point, string or
      general) per function
    * 256 function literal declarations plus unique functions calls per
      function
    * 256 types available per function
    * 256 unique native functions per function
    * 16384 integer values per function
    * 256 string values per function
    * 16384 floating-point values per function
    * 256 general values per function

`
