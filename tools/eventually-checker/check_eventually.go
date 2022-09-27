// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

const mode packages.LoadMode = packages.NeedName |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo

// funcCall contains information about a function call, including the function name and its position in a file
type funcCall struct {
	name string
	pos  token.Pos
}

// funcMap is a map of function names to functions called directly by the function
var funcMap = make(map[string][]funcCall)

// eventuallyMap is a map of locations of Eventually calls and the functions called directly by those Eventually calls
var eventuallyMap = make(map[token.Pos][]funcCall)

// if reportOnly is true, we always return a zero exit code
var reportOnly bool

// parseFlags sets up command line arg and flag parsing
func parseFlags() {
	flag.Usage = func() {
		help := "Usage of %s: [options] path\nScans all packages in path and outputs function calls that cause Eventually to exit prematurely\n\n"
		fmt.Fprintf(flag.CommandLine.Output(), help, os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolVar(&reportOnly, "report", false, "report on problems but always exits with a zero status code")
	flag.Parse()
}

// main loads the packages from the specified directories, analyzes the file sets, and displays information about
// calls that should not be called from Eventually
func main() {
	parseFlags()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	// load all packages from the specified directory
	fset, pkgs, err := loadPackages(flag.Args()[0])
	if err != nil {
		fmt.Fprintln(flag.CommandLine.Output(), err)
		os.Exit(1)
	}

	// analyze each package
	for _, pkg := range pkgs {
		analyze(pkg.Syntax)
	}

	// check for calls that should not be in Eventually blocks and display results
	results := checkForBadCalls()
	displayResults(results, fset, flag.CommandLine.Output())

	if len(results) > 0 && !reportOnly {
		os.Exit(1)
	}

	os.Exit(0)
}

// loadPackages loads the packages from the specified path and returns the FileSet, the slice of packages,
// and an error
func loadPackages(path string) (*token.FileSet, []*packages.Package, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{Tests: true, Fset: fset, Mode: mode, Dir: path}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, nil, err
	}
	return fset, pkgs, nil
}

// analyze analyzes all of the files in the file set, looking for calls to Fail & Expect inside of Eventually.
// We use ast.Inspect to walk the abstract syntax tree, searching for function declarations and function calls.
// If we find a call to Eventually, we record it in the map of Eventually calls, otherwise we record the
// call in funcMap.
func analyze(files []*ast.File) {
	for _, file := range files {
		var currentFuncDecl string
		var funcEnd token.Pos
		ast.Inspect(file, func(n ast.Node) bool {
			if n != nil && n.Pos() > funcEnd {
				// this node is after the current function decl end position, so reset
				currentFuncDecl = ""
				funcEnd = 0
			}

			pkg := file.Name.Name
			switch x := n.(type) {
			case *ast.FuncDecl:
				currentFuncDecl = getFuncDeclName(pkg, x)
				funcEnd = x.End()
			case *ast.CallExpr:
				name, pos := getNameAndPosFromCallExpr(x, file.Name.Name)

				if name != "" {
					if strings.HasSuffix(name, ".Eventually") {
						f, isAnonFunc := getEventuallyFuncName(pkg, x.Args)
						if isAnonFunc {
							inspectEventuallyAnonFunc(pos, x.Args[0], pkg, currentFuncDecl)
							// returning false tells the inspector there's no need to continue walking this
							// part of the tree
							return false
						}
						addCallToEventuallyMap(pos, f, pos)
					} else if currentFuncDecl != "" {
						if _, ok := funcMap[currentFuncDecl]; !ok {
							funcMap[currentFuncDecl] = make([]funcCall, 0)
						}
						funcMap[currentFuncDecl] = append(funcMap[currentFuncDecl], funcCall{name: name, pos: pos})
					}
				}
			}
			return true
		})
	}
}

// getFuncDeclName constructs a function name of the form pkg.func_name or pkg.type.func_name if the
// function is a method receiver
func getFuncDeclName(pkg string, funcDecl *ast.FuncDecl) string {
	baseFuncName := pkg

	if funcDecl.Recv != nil {
		// this function decl is a method receiver so include the type in the name
		recType := funcDecl.Recv.List[0].Type
		switch x := recType.(type) {
		case *ast.StarExpr:
			// pointer receiver
			baseFuncName = fmt.Sprintf("%s.%s", baseFuncName, x.X)
		case *ast.Ident:
			// value receiver
			baseFuncName = fmt.Sprintf("%s.%s", baseFuncName, x)
		}
	}

	return fmt.Sprintf("%s.%s", baseFuncName, funcDecl.Name.Name)
}

// inspectEventuallyAnonFunc finds all function calls in an anonymous function passed to Eventually
func inspectEventuallyAnonFunc(eventuallyPos token.Pos, node ast.Node, pkgName string, parent string) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			name, pos := getNameAndPosFromCallExpr(x, pkgName)
			addCallToEventuallyMap(eventuallyPos, name, pos)
		}
		return true
	})
}

// addCallToEventuallyMap adds a function name at the given position to the map of Eventually calls
func addCallToEventuallyMap(eventuallyPos token.Pos, funcName string, pos token.Pos) {
	if _, ok := eventuallyMap[eventuallyPos]; !ok {
		eventuallyMap[eventuallyPos] = make([]funcCall, 0)
	}
	eventuallyMap[eventuallyPos] = append(eventuallyMap[eventuallyPos], funcCall{name: funcName, pos: pos})
}

// getNameAndPosFromCallExpr gets the function name and position from an ast.CallExpr
func getNameAndPosFromCallExpr(expr *ast.CallExpr, pkgName string) (string, token.Pos) {
	switch x := expr.Fun.(type) {
	case *ast.Ident:
		// ast.Ident means the call is in the same package as the enclosing function declaration, so use the
		// package from the func decl
		name := pkgName + "." + x.Name
		pos := x.NamePos
		return name, pos
	case *ast.SelectorExpr:
		var pkg string
		var pos token.Pos
		if ident, ok := x.X.(*ast.Ident); ok {
			pos = ident.NamePos
			if ident.Obj != nil {
				// call is a method receiver so find the type of the receiver
				if valueSpec, ok := ident.Obj.Decl.(*ast.ValueSpec); ok {
					if selExpr, ok := valueSpec.Type.(*ast.SelectorExpr); ok {
						// type is not in the same package as the calling function
						if ident, ok = selExpr.X.(*ast.Ident); ok {
							pkg = ident.Name + "." + selExpr.Sel.Name + "."
						}
					} else if id, ok := valueSpec.Type.(*ast.Ident); ok {
						// type is in the same package as the caller
						pkg = pkgName + "." + id.Name + "."
					}
				}
			} else {
				pkg = ident.Name + "."
			}
		}
		name := pkg + x.Sel.Name
		return name, pos
	default:
		// ignore other function call types
		return "", 0
	}
}

// getEventuallyFuncName returns the name of the function (prefixed with package name) passed to
// Eventually and a boolean that will be true if an anonymous function is passed to Eventually
func getEventuallyFuncName(pkg string, args []ast.Expr) (string, bool) {
	if len(args) == 0 {
		panic("No args passed to Eventually call")
	}

	switch x := args[0].(type) {
	case *ast.FuncLit:
		return "", true
	case *ast.Ident:
		return pkg + "." + x.Name, false
	case *ast.SelectorExpr:
		var p = pkg + "."
		if ident, ok := x.X.(*ast.Ident); ok {
			p = ident.Name + "."
		}
		return p + x.Sel.Name, false
	default:
		panic(fmt.Sprintf("Unexpected AST node type found: %s", x))
	}
}

// checkForBadCalls searches all the functions called by Eventually functions looking for bad calls and
// returns a map of results, where the key has the position of the bad call and the values contains
// a slice with all of the positions of Eventually calls that call the function (directly or indirectly)
func checkForBadCalls() map[token.Pos][]token.Pos {
	var resultsMap = make(map[token.Pos][]token.Pos)

	for key, val := range eventuallyMap {
		for i := range val {
			if fc := findBadCall(&val[i], 0); fc != nil {
				if _, ok := resultsMap[fc.pos]; !ok {
					resultsMap[fc.pos] = make([]token.Pos, 0)
				}
				resultsMap[fc.pos] = append(resultsMap[fc.pos], key)
			}
		}
	}

	return resultsMap
}

// findBadCall does a depth-first search of function calls looking for calls to Fail or Expect - it returns
// nil if no bad calls are found, or information describing the call (name and file position) if a bad call is found
func findBadCall(fc *funcCall, depth int) *funcCall {
	// if there are any cycles in the call graph due to recursion, use depth value to prevent running forever
	if depth > 30 {
		return nil
	}

	if strings.HasSuffix(fc.name, ".Fail") || strings.HasSuffix(fc.name, ".Expect") {
		return fc
	}
	fn := funcMap[fc.name]
	for i := range fn {
		if childFuncCall := findBadCall(&fn[i], depth+1); childFuncCall != nil {
			return childFuncCall
		}
	}

	return nil
}

// displayResults outputs the analysis results
func displayResults(results map[token.Pos][]token.Pos, fset *token.FileSet, out io.Writer) {
	for key, val := range results {
		fmt.Fprintf(out, "eventuallyChecker: Fail/Expect at %s\n    called from Eventually at:\n", fset.PositionFor(key, true))
		for _, calls := range val {
			fmt.Fprintf(out, "        %s\n", fset.PositionFor(calls, true))
		}
		fmt.Fprintln(out)
	}
}
