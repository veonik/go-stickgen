package stickgen

import (
	"fmt"
	"bytes"
	"strings"
	"io/ioutil"
	"errors"
	"regexp"

	"github.com/tyler-sommer/stick"
	"github.com/tyler-sommer/stick/parse"
)

var notWordOrUnderscore = regexp.MustCompile(`[^\w_]`)

type renderer func()

type Generator struct {
	loader stick.Loader
	out *bytes.Buffer
	name string
	imports map[string]bool
	blocks map[string]renderer
	args []struct{ Name, Typ string }
	child bool
}

func (g *Generator) Generate(name string) (string, error) {
	tpl, err := g.loader.Load(name)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(tpl.Contents())
	if err != nil {
		return "", err
	}
	tree, err := parse.Parse(string(body))
	if err != nil {
		return "", err
	}
	if g.name == "" {
		g.name = string(notWordOrUnderscore.ReplaceAll([]byte(name), []byte("_")))
	}
	g.walk(tree.Root())
	return g.output(), nil
}

// NewGenerator creates a new code generator using the given Loader.
func NewGenerator(loader stick.Loader) *Generator {
	g := &Generator{
		loader: loader,
		name: "",
		out: &bytes.Buffer{},
		imports: make(map[string]bool),
		blocks: make(map[string]renderer),
		args: make([]struct{ Name, Typ string }, 0),
	}

	return g
}

func (g *Generator) output() string {
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

func (g *Generator) addImport(name string) {
	if _, ok := g.imports[name]; !ok {
		g.imports[name] = true
	}
}

func (g *Generator) addArg(name, typ string) {
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

func (g *Generator) walk(n parse.Node) error {
	switch node := n.(type) {
	case *parse.ModuleNode:
		if node.Parent != nil {
			if name, ok := g.evaluate(node.Parent.Tpl); ok {
				g.Generate(name)
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
		g.addImport("fmt")
		g.out.WriteString(fmt.Sprintf(`	// line %d, offset %d
	fmt.Print(%s)
`, node.Line, node.Offset, fmt.Sprintf("`%s`", node.Data)))
	case *parse.PrintNode:
		v, err := g.walkExpr(node.X)
		if err != nil {
			return err
		}
		g.addImport("fmt")
		g.addArg(v, "string")
		g.out.WriteString(fmt.Sprintf(`	// line %d, offset %d
	fmt.Print(%s)
`, node.Line, node.Offset, v))
	case *parse.BlockNode:
		g.addImport("fmt")
		g.blocks[node.Name] = func(g *Generator, node *parse.BlockNode) renderer {
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

func (g *Generator) evaluate(e parse.Expr) (string, bool) {
	switch expr := e.(type) {
	case *parse.StringExpr:
		return expr.Text, true
	}
	return "", false
}

func (g *Generator) walkExpr(e parse.Expr) (string, error) {
	switch expr := e.(type) {
	case *parse.NameExpr:
		return expr.Name, nil
	}
	return "", nil
}