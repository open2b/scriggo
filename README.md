# Scriggo

Scriggo is the fast and safe open-source Go runtime to embed the Go language as a scripting language in Go applications
and templates.

## Features

* Fast, the fastest pure Go language runtime.
* Secure by default. No access to packages unless explicitly enabled.
* Modern and powerful Template Engine with Go as scripting language.
* Native support for Markdown in templates.
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

Scriggo is also a modern and powerful template engine for Go, supporting inheritance,
macros, partials, imports and contextual autoescaping but most of all it uses the
Go language as the template scripting language.

```
{% extends "layout.html" %}
{% import "banners.html" %}
{% macro Body %}
    <ul>
      {% for _, product := range products %}
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

For a complete get started guide see the [Scriggo site](https://www.scriggo.com/).
