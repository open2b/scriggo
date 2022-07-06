
Scriggo command
---------------

The scriggo command is a command line tool that allows to:

  * serve templates with support for Markdown
  * initialize an interpreter for Go programs
  * generate the code for package importers


Serve templates
---------------

The Scriggo Serve command runs a web server and serves the template rooted at
the current directory. All Scriggo builtins are available in template files.
It is useful to learn Scriggo templates (https://scriggo.com/templates).

The basic Serve command takes this form:

  $ scriggo serve

It renders HTML and Markdown files based on file extension.

It does not require a Go installation.

For more details see the help with 'scriggo help serve' or visit
https://scriggo.com/scriggo-command#serve-a-template


Initialize an interpreter
-------------------------

The Scriggo Init command initializes an interpreter for Go programs.

Before using Init, download and install Go (https://go.dev/dl/).

For more details see the help with 'scriggo help init' or visit
https://scriggo.com/scriggo-command#initialize-an-interpreter


Generate a package importer
---------------------------

The Scriggo Import command generate the code for a package importer.
An importer is used by Scriggo to import a package when an "import"
declaration is executed.

The code for the importer is generated from the instructions in a
Scriggofile. The Scriggofile should be in a Go module.

Before using Import, download and install Go (https://go.dev/dl/).

For more details see the help with 'scriggo help import' or visit
https://scriggo.com/scriggo-command#generate-a-package-importer
