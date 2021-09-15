<img src="https://scriggo-site.pages.dev/images/scriggo.svg" alt="Scriggo" title="Scriggo" width="535px" style="max-width: 100%">

The world’s most powerful template engine and [Go](https://golang.org/) embeddable interpreter.

[![Go Reference](https://pkg.go.dev/badge/github.com/open2b/scriggo/.svg)](https://pkg.go.dev/github.com/open2b/scriggo/)

[Website](https://scriggo.com/) | [Get Started](https://scriggo.com/get-started) | [Documentation](https://scriggo.com/what-is-scriggo) | [Community](https://github.com/open2b/scriggo/discussions) | [Contributing](https://github.com/open2b/scriggo/blob/main/CONTRIBUTING.md)

## Features

* Fast, a very fast embeddable pure Go language interpreter.
* Modern and powerful template engine with Go as scripting language.
* Native support for Markdown in templates.
* Secure by default. No access to packages unless explicitly enabled.
* Easy to embed and to interop with any Go application.

## Get Started with Programs

Execute a Go program embedded in your application:

```go
package main

import "github.com/open2b/scriggo"

func main() {

    // src is the source code of the program to run.
    src := []byte(`
        package main

        func main() {
            println("Hello, World!")
        }
    `)

    // Create a file system with the file of the program to run.
    fsys := scriggo.Files{"main.go": src}

    // Build the program.
    program, err := scriggo.Build(fsys, nil)
    if err != nil {
        panic(err)
    }
 
    // Run the program.
    err = program.Run(nil)
    if err != nil {
        panic(err)
    }

}
```

## Get Started with Templates

Scriggo, in templates, supports inheritance, macros, partials, imports and contextual autoescaping but most of all it
uses the Go language as the template scripting language.

```
{% extends "layout.html" %}
{% import "banners.html" %}
{% macro Body %}
    <ul>
      {% for product in products %}
      <li><a href="{{ product.URL }}">{{ product.Name }}</a></li>
      {% end %}
    </ul>
    {{ render "pagination.html" }}
    {{ Banner() }}
{% end %}
```

Scriggo template files can be written in plain text, HTML, Markdown, CSS, JavaScript and JSON.

### Execute a Scriggo template in your application

```go
// Build and run a Scriggo template.
package main

import (
    "os"
    "github.com/open2b/scriggo"
)

func main() {

    // Content of the template file to run.
    content := []byte(`
    <!DOCTYPE html>
    <html>
    <head>Hello</head> 
    <body>
        {% who := "World" %}
        Hello, {{ who }}!
    </body>
    </html>
    `)

    // Create a file system with the file of the template to run.
    fsys := scriggo.Files{"index.html": content}

    // Build the template.
    template, err := scriggo.BuildTemplate(fsys, "index.html", nil)
    if err != nil {
        panic(err)
    }
 
    // Run the template and print it to the standard output.
    err = template.Run(os.Stdout, nil, nil)
    if err != nil {
        panic(err)
    }

}
```

For a complete get started guide see the [Scriggo site](https://scriggo.com/).

## Contributing

Want to help contribute to Scriggo? See [CONTRIBUTING.md](CONTRIBUTING.md).
