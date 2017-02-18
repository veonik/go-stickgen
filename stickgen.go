package stickgen

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/tyler-sommer/stick"
	"github.com/tyler-sommer/stick/parse"
)

var notWord = regexp.MustCompile(`[^\w]|[_]`)

func titleize(in string) string {
	return strings.Replace(strings.Title(notWord.ReplaceAllString(in, " ")), " ", "", -1)
}

type renderer func()

type evaluatedExpr struct {
	body          string
	isFunction    bool
	hasError      bool
	resultantName string
}

// A Generator handles generating Go code from Twig templates.
type Generator struct {
	loader  stick.Loader
	out     *bytes.Buffer
	name    string
	imports map[string]bool
	blocks  map[string]renderer
	args    map[string]bool
	root    bool
	stack   []string
	tabs    int
}

// Generate parses the given template and outputs the generated code.
func (g *Generator) Generate(name string) (string, error) {
	err := g.generate(name)
	if err != nil {
		return "", err
	}
	return g.output(), nil
}

// NewGenerator creates a new code generator using the given Loader.
func NewGenerator(loader stick.Loader) *Generator {
	g := &Generator{
		loader:  loader,
		name:    "",
		out:     &bytes.Buffer{},
		imports: map[string]bool{},
		blocks:  make(map[string]renderer),
		args:    make(map[string]bool),
		root:    true,
		stack:   make([]string, 0),
		tabs:    1,
	}

	return g
}

func (g *Generator) indent() string {
	return strings.Repeat("	", g.tabs)
}

func (g *Generator) generate(name string) error {
	tpl, err := g.loader.Load(name)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(tpl.Contents())
	if err != nil {
		return err
	}
	tree, err := parse.Parse(string(body))
	if err != nil {
		return err
	}
	g.name = name
	g.stack = append(g.stack, name)
	g.root = len(g.stack) == 1
	defer func() {
		g.name, g.stack = g.stack[len(g.stack)-1], g.stack[:len(g.stack)-1]
		g.root = len(g.stack) == 1
	}()
	g.walk(tree.Root())
	return nil
}

func (g *Generator) output() string {
	body := g.out.String()
	funcs := make([]string, 0)
	for _, block := range g.blocks {
		g.out.Reset()
		block()
		funcs = append(funcs, g.out.String())
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

%s

func template%s(ctx map[string]stick.Value) {
%s}
`, strings.Join(imports, "\n	"), strings.Join(funcs, "\n"), titleize(g.name), body)
}

func (g *Generator) addImport(name string) {
	if _, ok := g.imports[name]; !ok {
		g.imports[name] = true
	}
}

func (g *Generator) walk(n parse.Node) error {
	switch node := n.(type) {
	case *parse.ModuleNode:
		if node.Parent != nil {
			if name, ok := g.evaluate(node.Parent.Tpl); ok {
				err := g.generate(name)
				if err != nil {
					return err
				}
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
	case *parse.IncludeNode:
		if name, ok := g.evaluate(node.Tpl); ok {
			err := g.generate(name)
			if err != nil {
				return err
			}
		} else {
			// TODO: Handle more than just string literals
			return errors.New("Unable to evaluate extends reference")
		}
	case *parse.TextNode:
		g.addImport("fmt")
		g.out.WriteString(fmt.Sprintf(`%s// line %d, offset %d in %s
%sfmt.Print(%s)
`, g.indent(), node.Line, node.Offset, g.name, g.indent(), fmt.Sprintf("`%s`", node.Data)))
	case *parse.PrintNode:
		v, err := g.walkExpr(node.X)
		if err != nil {
			return err
		}
		g.addImport("fmt")
		g.out.WriteString(fmt.Sprintf(`%s// line %d, offset %d in %s
`, g.indent(), node.Line, node.Offset, g.name))
		if v.isFunction {
			// TODO: The goggles, they do nothing!
			g.out.WriteString(fmt.Sprintf(`%s{
`, g.indent()))
			g.tabs++
			g.out.WriteString(fmt.Sprintf(`%s%s
`, g.indent(), v.body))
			if v.hasError {
				g.out.WriteString(fmt.Sprintf(`%sif err == nil {
`, g.indent()))
				g.tabs++
				g.out.WriteString(fmt.Sprintf(`%sfmt.Print(%s)
`, g.indent(), v.resultantName))
				g.tabs--
				g.out.WriteString(fmt.Sprintf(`%s}
`, g.indent()))
			} else {
				g.out.WriteString(fmt.Sprintf(`%sfmt.Print(%s)
`, g.indent(), v.resultantName))
			}
			g.tabs--
			g.out.WriteString(fmt.Sprintf(`%s}
`, g.indent()))
		} else {
			g.out.WriteString(fmt.Sprintf(`%sfmt.Print(%s)
`, g.indent(), v.resultantName))
		}

	case *parse.BlockNode:
		g.addImport("fmt")
		g.blocks[node.Name] = func(g *Generator, node *parse.BlockNode, rootName string) renderer {
			// TODO: Wow, I don't know about all this.
			return func() {
				g.out.WriteString(fmt.Sprintf(`func block%s%s(ctx map[string]stick.Value) {
`, titleize(rootName), titleize(node.Name)))
				g.walk(node.Body)
				g.out.WriteString(`}`)
			}
		}(g, node, g.stack[0])
		if !g.root {
			g.out.WriteString(fmt.Sprintf(`%s// line %d, offset %d in %s
%sblock%s%s(ctx)
`, g.indent(), node.Line, node.Offset, g.name, g.indent(), titleize(g.stack[0]), titleize(node.Name)))
		}
	case *parse.ForNode:
		name, err := g.walkExpr(node.X)
		if err != nil {
			return err
		}
		key := "_"
		if node.Key != "" {
			key = node.Key
			g.args[key] = true
		}
		val := node.Val
		g.args[val] = true
		g.addImport("github.com/tyler-sommer/stick")
		g.out.WriteString(fmt.Sprintf(`%s// line %d, offset %d in %s
%sstick.Iterate(%s, func(%s, %s stick.Value, loop stick.Loop) (brk bool, err error) {
`, g.indent(), node.Line, node.Pos, g.name, g.indent(), name.resultantName, key, val))
		g.tabs++
		g.walk(node.Body)
		delete(g.args, val)
		delete(g.args, key)
		g.out.WriteString(fmt.Sprintf(`%sreturn true, nil
`, g.indent()))
		g.tabs--
		g.out.WriteString(fmt.Sprintf(`%s})
`, g.indent()))
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

func newNameExpr(name string) evaluatedExpr {
	return evaluatedExpr{body: name, resultantName: name, isFunction: false, hasError: false}
}

var emptyExpr = evaluatedExpr{body: "", resultantName: "", isFunction: false, hasError: false}

func (g *Generator) walkExpr(e parse.Expr) (evaluatedExpr, error) {
	switch expr := e.(type) {
	case *parse.NameExpr:
		if _, ok := g.args[expr.Name]; ok {
			return newNameExpr(expr.Name), nil
		}
		return newNameExpr("ctx[\"" + expr.Name + "\"]"), nil
	case *parse.StringExpr:
		return newNameExpr(expr.Text), nil
	case *parse.GetAttrExpr:
		if len(expr.Args) > 0 {
			return emptyExpr, errors.New("Method calls are unsupported.")
		}
		attr, err := g.walkExpr(expr.Attr)
		if err != nil {
			return emptyExpr, err
		}
		name, err := g.walkExpr(expr.Cont)
		if err != nil {
			return emptyExpr, err
		}
		g.addImport("github.com/tyler-sommer/stick")
		return evaluatedExpr{body: `val, err := stick.GetAttr(` + name.resultantName + `, "` + attr.resultantName + `")`, resultantName: "val", isFunction: true, hasError: true}, nil
	}
	return emptyExpr, nil
}
