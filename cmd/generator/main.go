package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
)

type Fragment struct {
	SQL  string
	Args []string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run generator.go input.go output.go")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	fragments := []Fragment{}
	var isCombined bool

	ast.Inspect(node, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.BinaryExpr:
			if expr.Op == token.ADD {
				isCombined = true
			}
		case *ast.CallExpr:
			if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == "doobie" && sel.Sel.Name == "fragment" {
					if len(expr.Args) > 0 {
						if lit, ok := expr.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							fragments = append(fragments, processFragment(lit.Value))
						}
					}
				}
			}
		}
		return true
	})

	generateOutput(outputFile, fragments, isCombined)
}

func processFragment(s string) Fragment {
	s = strings.Trim(s, "`")
	interpolationRegex := regexp.MustCompile(`\${(.*?)}`)
	matches := interpolationRegex.FindAllStringSubmatch(s, -1)

	args := make([]string, len(matches))
	for i, match := range matches {
		args[i] = match[1]
		s = strings.Replace(s, match[0], "?", 1)
	}

	return Fragment{SQL: s, Args: args}
}

func generateOutput(outputFile string, fragments []Fragment, isCombined bool) {
	f, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(f, "package main")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "import \"github.com/your-repo/doobie\"")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "func generateQuery(q Query) doobie.Fragment {")

	if isCombined {
		fmt.Fprintln(f, "    return doobie.CombineFragments(")
		for i, frag := range fragments {
			fmt.Fprintln(f, "        doobie.Fragment{")
			fmt.Fprintf(f, "            SQL: `%s`,\n", frag.SQL)
			fmt.Fprintln(f, "            Args: []interface{}{")
			for _, arg := range frag.Args {
				fmt.Fprintf(f, "                %s,\n", arg)
			}
			fmt.Fprintln(f, "            },")
			fmt.Fprint(f, "        }")
			if i < len(fragments)-1 {
				fmt.Fprintln(f, ",")
			} else {
				fmt.Fprintln(f)
			}
		}
		fmt.Fprintln(f, "    )")
	} else {
		fmt.Fprintln(f, "    return doobie.Fragment{")
		fmt.Fprint(f, "        SQL: `")
		for _, frag := range fragments {
			fmt.Fprint(f, frag.SQL)
		}
		fmt.Fprintln(f, "`,")
		fmt.Fprintln(f, "        Args: []interface{}{")
		for _, frag := range fragments {
			for _, arg := range frag.Args {
				fmt.Fprintf(f, "            %s,\n", arg)
			}
		}
		fmt.Fprintln(f, "        },")
		fmt.Fprintln(f, "    }")
	}

	fmt.Fprintln(f, "}")
}
