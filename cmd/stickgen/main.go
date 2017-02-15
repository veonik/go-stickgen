package main

import (
	"github.com/tyler-sommer/stick"
	"fmt"
	"github.com/tyler-sommer/go-stickgen"
)

func main() {
	loader := &stick.MemoryLoader{
		Templates: map[string]string{
			"layout.twig": `Hello, {% block name %}{% endblock %}!`,
			"test.twig": `{% extends 'layout.twig' %}{% block name %}World{% endblock %}`,
		},
	}

	g := stickgen.NewGenerator(loader)
	output, err := g.Generate("test.twig")
	if err != nil {
		panic(err)
	}

	fmt.Println(output)
}