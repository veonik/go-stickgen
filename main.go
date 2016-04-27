package main

import (
	"github.com/tyler-sommer/stick/parse"
	"fmt"
	"os"
	"bytes"
	"strings"
)

func main() {
	tpl := `Hello, {{ name }}!`

	tree, err := parse.Parse(tpl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	g := newGenerator()
	g.walk(tree.Root())

	fmt.Println(g.Output())
}

type generator struct {
	out *bytes.Buffer
	imports map[string]bool
	args []struct{ Name, Typ string }
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

func newGenerator() *generator {
	g := &generator{&bytes.Buffer{}, make(map[string]bool), make([]struct{ Name, Typ string }, 0)}

	return g
}

func (g *generator) Output() string {
	args := make([]string, 0)
	for _, v := range g.args {
		args = append(args, fmt.Sprintf("%s %s", v.Name, v.Typ))
	}
	imports := make([]string, 0)
	for v, _ := range g.imports {
		imports = append(imports, fmt.Sprintf(`"%s"`, v))
	}

	return fmt.Sprintf(`
package main

import (
	%s
)

func template(%s) {
%s
}
`, strings.Join(imports, "\n	"), strings.Join(args, ", "), g.out.String())
}

func (g *generator) walk(n parse.Node) error {
	switch node := n.(type) {
	case *parse.ModuleNode:
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
	fmt.Print("%s")
`, node.Line, node.Offset, node.Text()))
	case *parse.PrintNode:
		v, err := g.walkExpr(node.Expr())
		if err != nil {
			return err
		}
		g.Import("fmt")
		g.Arg(v, "string")
		g.out.WriteString(fmt.Sprintf(`	// line %d, offset %d
	fmt.Print(%s)
`, node.Line, node.Offset, v))
	}
	return nil
}

func (g *generator) walkExpr(e parse.Expr) (string, error) {
	switch expr := e.(type) {
	case *parse.NameExpr:
		return expr.Name(), nil
	}
	return "", nil
}