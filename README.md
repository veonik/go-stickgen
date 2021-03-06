# go-stickgen

[![GoDoc](https://godoc.org/github.com/veonik/go-stickgen?status.svg)](https://godoc.org/github.com/veonik/go-stickgen)

Generates Go code for stick templates.

> This project is currently in a proof-of-concept state, and is under early development.

Installation
------------

Install the stickgen library and command with:

```
go get -u github.com/veonik/go-stickgen/...
```

Usage
-----

```
Usage: stickgen [-path <templates>] [-out <generated>] <glob>
  -out string
    	Output path (default "./generated")
  -path string
    	Path to templates (default ".")
```

### Usage as a library

Below is a simple example that uses the stickgen `Generator`.

```go
package main

import (
	"fmt"

	"github.com/tyler-sommer/stick"
	"github.com/veonik/go-stickgen"
)

func main() {
	loader := &stick.MemoryLoader{
		Templates: map[string]string{
			"layout.twig": `Hello, {% block name %}{% endblock %}!`,
			"test.twig":   `{% extends 'layout.twig' %}{% block name %}World{% endblock %}`,
		},
	}

	g := stickgen.NewGenerator("views", loader)
	output, err := g.Generate("test.twig")
	if err != nil {
		fmt.Printf("An error occured: %s", err.Error())
		return
	}

	fmt.Println(output)
	// Output:
	// // Code generated by stickgen.
	// // DO NOT EDIT!
	//
	// package views
	//
	// import (
	// 	"github.com/tyler-sommer/stick"
	// 	"io"
	// 	"fmt"
	// )
	//
	// func blockTestTwigName(env *stick.Env, output io.Writer, ctx map[string]stick.Value) {
	// 	// line 1, offset 43 in test.twig
	// 	fmt.Fprint(output, `World`)
	// }
	//
	// func TemplateTestTwig(env *stick.Env, output io.Writer, ctx map[string]stick.Value) {
	// 	// line 1, offset 0 in layout.twig
	// 	fmt.Fprint(output, `Hello, `)
	// 	// line 1, offset 10 in layout.twig
	// 	blockTestTwigName(env, output, ctx)
	// 	// line 1, offset 37 in layout.twig
	// 	fmt.Fprint(output, `!`)
	// }
}
```

