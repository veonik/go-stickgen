package main

import (
	"github.com/tyler-sommer/stick/parse"
	"fmt"
	"os"
	"bytes"
	"strings"
	"github.com/tyler-sommer/stick"
	"io/ioutil"
	"errors"
	"regexp"
)

var underscorize = regexp.MustCompile(`[^\w_]`)

func main() {
	loader := &stick.MemoryLoader{
		Templates: map[string]string{
			"layout.twig": `Hello, {% block name %}{% endblock %}!`,
			"test.twig": `{% extends 'layout.twig' %}{% block name %}World{% endblock %}`,
		},
	}

	g := newGenerator(loader)
	g.parse("test.twig")

	fmt.Println(g.Output())
}

type renderer func()

type generator struct {
	loader stick.Loader
	out *bytes.Buffer
	name string
	imports map[string]bool
	blocks map[string]renderer
	args []struct{ Name, Typ string }
	child bool
}

func (g *generator) parse(name string) {
	tpl, err := g.loader.Load(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(tpl.Contents())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	tree, err := parse.Parse(string(body))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if g.name == "" {
		g.name = string(underscorize.ReplaceAll([]byte(name), []byte("_")))
	}
	g.walk(tree.Root())
}

func (g *generator) Import(name string) {
	if _, ok := g.imports[name]; !ok {
		g.imports[name] = true
	}
}

func (g *generator) Arg(name, typ string) {
	exists := false
	for _, v := range g.args {
		if v.Name == name {
			exists = true
			break
		}
	}
	if !exists {
		g.args = append(g.args, struct{ Name, Typ string }{name, typ})
	}
}

func newGenerator(loader stick.Loader) *generator {
	g := &generator{
		loader: loader,
		name: "",
		out: &bytes.Buffer{},
		imports: make(map[string]bool),
		blocks: make(map[string]renderer),
		args: make([]struct{ Name, Typ string }, 0),
	}

	return g
}

func (g *generator) Output() string {
	args := make([]string, len(g.args))
	for _, v := range g.args {
		args = append(args, fmt.Sprintf("%s %s", v.Name, v.Typ))
	}
	imports := make([]string, len(g.imports))
	for v, _ := range g.imports {
		imports = append(imports, fmt.Sprintf(`"%s"`, v))
	}
	body := g.out.String()
	funcs := make([]string, len(g.blocks))
	for _, block := range g.blocks {
		g.out.Reset()
		block()
		funcs = append(funcs, g.out.String())
	}


	return fmt.Sprintf(`
package main

import (
	%s
)

%s

func template_%s(%s) {
%s}
`, strings.Join(imports, "\n	"), strings.Join(funcs, "\n"), g.name, strings.Join(args, ", "), body)
}

func (g *generator) walk(n parse.Node) error {
	switch node := n.(type) {
	case *parse.ModuleNode:
		if node.Parent != nil {
			if name, ok := g.evaluate(node.Parent.Tpl); ok {
				g.parse(name)
				g.child = true
			} else {
				// TODO: Handle more than just string literals
				return errors.New("Unable to evaluate extends reference")
			}
		}
		return g.walk(node.BodyNode)
	case *parse.BodyNode:
		for _, child := range node.All() {
			err := g.walk(child)
			if err != nil {
				return err
			}
		}
	case *parse.TextNode:
		g.Import("fmt")
		g.out.WriteString(fmt.Sprintf(`	// line %d, offset %d
	fmt.Print(%s)
`, node.Line, node.Offset, fmt.Sprintf("`%s`", node.Data)))
	case *parse.PrintNode:
		v, err := g.walkExpr(node.X)
		if err != nil {
			return err
		}
		g.Import("fmt")
		g.Arg(v, "string")
		g.out.WriteString(fmt.Sprintf(`	// line %d, offset %d
	fmt.Print(%s)
`, node.Line, node.Offset, v))
	case *parse.BlockNode:
		g.Import("fmt")
		g.blocks[node.Name] = func(g *generator, node *parse.BlockNode) renderer {
			// TODO: Wow, I don't know about all this.
			return func() {
				g.out.WriteString(fmt.Sprintf(`func block_%s() {
`, node.Name))
				g.walk(node.Body)
				g.out.WriteString(`}`)
			}
		}(g, node)
		if !g.child {
			g.out.WriteString(fmt.Sprintf(`	// line %d, offset %d
	block_%s()
`, node.Line, node.Offset, node.Name))
		}
	}
	return nil
}

func (g *generator) evaluate(e parse.Expr) (string, bool) {
	switch expr := e.(type) {
	case *parse.StringExpr:
		return expr.Text, true
	}
	return "", false
}

func (g *generator) walkExpr(e parse.Expr) (string, error) {
	switch expr := e.(type) {
	case *parse.NameExpr:
		return expr.Name, nil
	}
	return "", nil
}