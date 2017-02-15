# go-stickgen

[![GoDoc](https://godoc.org/github.com/tyler-sommer/go-stickgen?status.svg)](https://godoc.org/github.com/tyler-sommer/go-stickgen)

Generates Go code for stick templates.

> This project is currently in a proof-of-concept state, and is under early development.

Installation
------------

Install the stickgen library with:

```
go get github.com/tyler-sommer/go-stickgen
```

Usage
-----

Below is a simple example that uses stickgen.

```go
package main

import (
	"fmt"

	"github.com/tyler-sommer/go-stickgen"
	"github.com/tyler-sommer/stick"
)

func main() {
	loader := &stick.MemoryLoader{
		Templates: map[string]string{
			"layout.twig": `Hello, {% block name %}{% endblock %}!`,
			"test.twig":   `{% extends 'layout.twig' %}{% block name %}World{% endblock %}`,
		},
	}

	g := stickgen.NewGenerator(loader)
	output, err := g.Generate("test.twig")
	if err != nil {
		fmt.Printf("An error occured: %s", err.Error())
		return
	}

	fmt.Println(output)
}
```

