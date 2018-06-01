/*

Command stickgen generates Go code from a Stick/Twig template.


	$ go get -u github.com/veonik/go-stickgen/...

Stickgen takes an input path where views are stored, an output path for
generated files, and a glob for matching templates.

	Usage: stickgen [-path <templates>] [-out <generated>] <glob>
	  -out string
	    	Output path (default "./generated")
	  -path string
	    	Path to templates (default ".")
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/tyler-sommer/stick"
	"github.com/veonik/go-stickgen"
)

var path = flag.String("path", ".", "Path to templates")
var out = flag.String("out", "./generated", "Output path")

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: stickgen [-path <templates>] [-out <generated>] <glob>")
		flag.PrintDefaults()
	}
	flag.Parse()
	loader := stick.NewFilesystemLoader(*path)

	if flag.NArg() == 0 {
		fmt.Println("stickgen: expects one arg, glob to generate")
		return
	}
	err := os.MkdirAll(*out, 0755)
	if err != nil {
		fmt.Printf("stickgen: output path is not a directory: %s\n", *out)
	}
	files, err := filepath.Glob(filepath.Join(*path, flag.Arg(0)))
	if err != nil {
		fmt.Printf("stickgen: unable to glob inputs: %s\n", err.Error())
		return
	}
	outfiles := make([]string, len(files))
	for i, file := range files {
		tpl, err := filepath.Rel(*path, file)
		if err != nil {
			fmt.Printf("stickgen: unable to locate input file: %s\n", err)
			return
		}
		outfile := filepath.Join(*out, tpl) + ".go"
		dirName := filepath.Dir(outfile)
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			fmt.Printf("stickgen: output path is not a directory: %s\n", *out)
		}
		fmt.Printf("Generating %s as %s\n", file, outfile)
		outfiles[i] = outfile
		g := stickgen.NewGenerator(filepath.Base(dirName), loader)
		output, err := g.Generate(tpl)
		if err != nil {
			fmt.Printf("stickgen: unable to generate code: %s\n", err)
			return
		}
		err = ioutil.WriteFile(outfile, []byte(output), 0644)
		if err != nil {
			fmt.Printf("stickgen: unable to write output: %s\n", err)
		}
	}

}
