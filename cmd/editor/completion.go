//go:build js

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"syscall/js"
)

type completionItem struct {
	Label      string `json:"label"`
	InsertText string `json:"insertText"`
	Kind       int    `json:"kind"`
	Detail     string `json:"detail"`
}

const (
	completionKindKeyword  = 14
	completionKindFunction = 1
	completionKindClass    = 5
	completionKindVariable = 4
	completionKindConstant = 9
	completionKindModule   = 6
	completionKindField    = 7
	completionKindMethod   = 2
)

var goKeywords = []completionItem{
	{"break", "break", completionKindKeyword, ""},
	{"case", "case ", completionKindKeyword, ""},
	{"chan", "chan ", completionKindKeyword, ""},
	{"const", "const ", completionKindKeyword, ""},
	{"continue", "continue", completionKindKeyword, ""},
	{"default", "default", completionKindKeyword, ""},
	{"defer", "defer ", completionKindKeyword, ""},
	{"else", "else ", completionKindKeyword, ""},
	{"fallthrough", "fallthrough", completionKindKeyword, ""},
	{"for", "for ", completionKindKeyword, ""},
	{"func", "func ", completionKindKeyword, ""},
	{"go", "go ", completionKindKeyword, ""},
	{"goto", "goto ", completionKindKeyword, ""},
	{"if", "if ", completionKindKeyword, ""},
	{"import", "import ", completionKindKeyword, ""},
	{"interface", "interface ", completionKindKeyword, ""},
	{"map", "map[", completionKindKeyword, ""},
	{"package", "package ", completionKindKeyword, ""},
	{"range", "range ", completionKindKeyword, ""},
	{"return", "return ", completionKindKeyword, ""},
	{"select", "select ", completionKindKeyword, ""},
	{"struct", "struct ", completionKindKeyword, ""},
	{"switch", "switch ", completionKindKeyword, ""},
	{"type", "type ", completionKindKeyword, ""},
	{"var", "var ", completionKindKeyword, ""},
}

var goBuiltins = []completionItem{
	{"bool", "bool", completionKindClass, "built-in type"},
	{"byte", "byte", completionKindClass, "built-in type (alias for uint8)"},
	{"complex64", "complex64", completionKindClass, "built-in type"},
	{"complex128", "complex128", completionKindClass, "built-in type"},
	{"error", "error", completionKindClass, "built-in interface"},
	{"float32", "float32", completionKindClass, "built-in type"},
	{"float64", "float64", completionKindClass, "built-in type"},
	{"int", "int", completionKindClass, "built-in type"},
	{"int8", "int8", completionKindClass, "built-in type"},
	{"int16", "int16", completionKindClass, "built-in type"},
	{"int32", "int32", completionKindClass, "built-in type"},
	{"int64", "int64", completionKindClass, "built-in type"},
	{"rune", "rune", completionKindClass, "built-in type (alias for int32)"},
	{"string", "string", completionKindClass, "built-in type"},
	{"uint", "uint", completionKindClass, "built-in type"},
	{"uint8", "uint8", completionKindClass, "built-in type"},
	{"uint16", "uint16", completionKindClass, "built-in type"},
	{"uint32", "uint32", completionKindClass, "built-in type"},
	{"uint64", "uint64", completionKindClass, "built-in type"},
	{"uintptr", "uintptr", completionKindClass, "built-in type"},
	{"any", "any", completionKindClass, "built-in type (alias for interface{})"},
	{"comparable", "comparable", completionKindClass, "built-in interface"},
	{"true", "true", completionKindConstant, "boolean constant"},
	{"false", "false", completionKindConstant, "boolean constant"},
	{"nil", "nil", completionKindConstant, "nil pointer/interface/slice/map/func/chan"},
	{"iota", "iota", completionKindConstant, "iota constant generator"},
	{"append", "append()", completionKindFunction, "built-in function"},
	{"cap", "cap()", completionKindFunction, "built-in function"},
	{"close", "close()", completionKindFunction, "built-in function"},
	{"copy", "copy()", completionKindFunction, "built-in function"},
	{"delete", "delete()", completionKindFunction, "built-in function"},
	{"imag", "imag()", completionKindFunction, "built-in function"},
	{"len", "len()", completionKindFunction, "built-in function"},
	{"make", "make()", completionKindFunction, "built-in function"},
	{"new", "new()", completionKindFunction, "built-in function"},
	{"panic", "panic()", completionKindFunction, "built-in function"},
	{"print", "print()", completionKindFunction, "built-in function"},
	{"println", "println()", completionKindFunction, "built-in function"},
	{"real", "real()", completionKindFunction, "built-in function"},
	{"recover", "recover()", completionKindFunction, "built-in function"},
}

func (j *jsEditor) getCompletions(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return js.ValueOf([]interface{}{})
	}
	content := args[0].String()
	line := args[1].Int()
	column := args[2].Int()

	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, j.filePath, content, parser.AllErrors)

	// Collect completions
	var items []completionItem

	// Find the prefix before cursor to determine context
	lines := strings.Split(content, "\n")
	var prefix string
	if line > 0 && line <= len(lines) {
		lineStr := lines[line-1]
		if column > 0 && column <= len(lineStr) {
			prefix = strings.TrimSpace(lineStr[:column-1])
		}
	}

	isDotCompletion := strings.HasSuffix(prefix, ".")
	dotPrefix := strings.TrimSuffix(prefix, ".")

	if isDotCompletion && f != nil {
		items = append(items, j.dotCompletion(f, fset, dotPrefix)...)
	} else if strings.Contains(prefix, ".") && f != nil {
		// Mid-dot completion: user typed "fmt.P" — still resolve package exports
		dotParts := strings.SplitN(prefix, ".", 2)
		items = append(items, j.dotCompletion(f, fset, strings.TrimSpace(dotParts[0]))...)
	}
	if !isDotCompletion {
		// Keywords
		items = append(items, goKeywords...)
		// Builtins
		items = append(items, goBuiltins...)
		// Package-level identifiers in current file
		if f != nil {
			items = append(items, fileDeclarations(f)...)
		}
		// Imported package names for quick access
		if f != nil {
			items = append(items, importedPkgRefs(f)...)
		}
	}

	// Filter by prefix
	items = filterByPrefix(items, lastWord(prefix))

	// Deduplicate
	items = dedup(items)

	// Convert to JS array
	return completionItemsToJS(items)
}

func lastWord(prefix string) string {
	// For dot completions, get the word after the last dot
	if idx := strings.LastIndex(prefix, "."); idx >= 0 {
		return prefix[idx+1:]
	}
	// For normal completions, get the last word
	fields := strings.Fields(prefix)
	if len(fields) > 0 {
		last := fields[len(fields)-1]
		// Remove trailing punctuation
		last = strings.TrimRight(last, "({[ ")
		return last
	}
	return ""
}

func filterByPrefix(items []completionItem, prefix string) []completionItem {
	if prefix == "" {
		return items
	}
	prefix = strings.ToLower(prefix)
	var filtered []completionItem
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item.Label), prefix) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func dedup(items []completionItem) []completionItem {
	seen := make(map[string]bool)
	var result []completionItem
	for _, item := range items {
		if !seen[item.Label] {
			seen[item.Label] = true
			result = append(result, item)
		}
	}
	return result
}

func completionItemsToJS(items []completionItem) js.Value {
	jsItems := make([]interface{}, len(items))
	for i, item := range items {
		jsItems[i] = map[string]interface{}{
			"label":      item.Label,
			"insertText": item.InsertText,
			"kind":       item.Kind,
			"detail":     item.Detail,
		}
	}
	return js.ValueOf(jsItems)
}

// fileDeclarations returns all top-level declarations (functions, vars, types, consts)
func fileDeclarations(f *ast.File) []completionItem {
	var items []completionItem
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					kind := completionKindClass
					if _, ok := s.Type.(*ast.InterfaceType); ok {
						kind = completionKindClass
					}
					items = append(items, completionItem{s.Name.Name, s.Name.Name, kind, "type declaration"})
				case *ast.ValueSpec:
					for _, name := range s.Names {
						k := completionKindVariable
						if d.Tok == token.CONST {
							k = completionKindConstant
						}
						items = append(items, completionItem{name.Name, name.Name, k, "declaration"})
					}
				}
			}
		case *ast.FuncDecl:
			if d.Name != nil {
				sig := d.Name.Name + "("
				if d.Type.Params != nil {
					params := make([]string, len(d.Type.Params.List))
					for i, p := range d.Type.Params.List {
						params[i] = typeExprString(p.Type)
					}
					sig += strings.Join(params, ", ")
				}
				sig += ")"
				items = append(items, completionItem{d.Name.Name, d.Name.Name, completionKindFunction, sig})
			}
		}
	}
	return items
}

func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeExprString(t.Elt)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return ""
	}
}

// importedPkgRefs returns the package alias (e.g., "fmt") for quick access
func importedPkgRefs(f *ast.File) []completionItem {
	var items []completionItem
	for _, imp := range f.Imports {
		pkgPath := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			items = append(items, completionItem{imp.Name.Name, imp.Name.Name, completionKindModule, pkgPath})
		} else {
			// Default package name = last element of import path
			parts := strings.Split(pkgPath, "/")
			name := parts[len(parts)-1]
			items = append(items, completionItem{name, name, completionKindModule, pkgPath})
		}
	}
	return items
}

// knownPkgCompletions returns common standard library packages for import completion
func knownPkgCompletions() []completionItem {
	return []completionItem{
		{"fmt", "fmt", completionKindModule, "fmt"},
		{"os", "os", completionKindModule, "os"},
		{"strings", "strings", completionKindModule, "strings"},
		{"strconv", "strconv", completionKindModule, "strconv"},
		{"time", "time", completionKindModule, "time"},
		{"math", "math", completionKindModule, "math"},
		{"net/http", "net/http", completionKindModule, "net/http"},
		{"encoding/json", "encoding/json", completionKindModule, "encoding/json"},
		{"io", "io", completionKindModule, "io"},
		{"bytes", "bytes", completionKindModule, "bytes"},
		{"sort", "sort", completionKindModule, "sort"},
		{"sync", "sync", completionKindModule, "sync"},
		{"context", "context", completionKindModule, "context"},
		{"errors", "errors", completionKindModule, "errors"},
		{"log", "log", completionKindModule, "log"},
		{"flag", "flag", completionKindModule, "flag"},
		{"path/filepath", "path/filepath", completionKindModule, "path/filepath"},
		{"regexp", "regexp", completionKindModule, "regexp"},
		{"testing", "testing", completionKindModule, "testing"},
	}
}

// dotCompletion handles completions after a dot (e.g., "fmt.P")
func (j *jsEditor) dotCompletion(f *ast.File, fset *token.FileSet, dotPrefix string) []completionItem {
	dotPrefix = strings.TrimSpace(dotPrefix)
	if dotPrefix == "" {
		return nil
	}

	// Resolve dot prefix to an imported package
	for _, imp := range f.Imports {
		pkgPath := strings.Trim(imp.Path.Value, "\"")
		pkgName := pkgPath
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(pkgPath, "/")
			pkgName = parts[len(parts)-1]
		}
		if pkgName == dotPrefix {
			return j.resolvePackageExports(pkgPath)
		}
	}

	// Check for local variable dot completion (v2: type-check based)
	// For now, try parsing declarations for struct types
	return j.localDotCompletion(f, dotPrefix)
}

func (j *jsEditor) resolvePackageExports(pkgPath string) []completionItem {
	// For well-known stdlib packages, use the hardcoded list directly.
	// Reading from the virtual filesystem (Go toolchain overlay) is unreliable
	// in the WASM child process.
	if items := wellKnownExports(pkgPath); items != nil {
		return items
	}

	// For third-party packages, try to read from GOMODCACHE in the virtual FS.
	dir := j.findPackageDir(pkgPath)
	if dir == "" {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	fset := token.NewFileSet()
	exported := make(map[string]string) // name -> kind

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		parsed, err := parser.ParseFile(fset, entry.Name(), src, parser.PackageClauseOnly)
		if err != nil {
			parsed, _ = parser.ParseFile(fset, entry.Name(), src, parser.AllErrors)
		}
		if parsed == nil {
			continue
		}
		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							exported[s.Name.Name] = "type"
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() {
								k := "var"
								if d.Tok == token.CONST {
									k = "const"
								}
								exported[name.Name] = k
							}
						}
					}
				}
			case *ast.FuncDecl:
				if d.Name != nil && d.Name.IsExported() {
					exported[d.Name.Name] = "func"
				}
			}
		}
	}

	var items []completionItem
	for name, kind := range exported {
		k := completionKindFunction
		switch kind {
		case "type":
			k = completionKindClass
		case "var":
			k = completionKindVariable
		case "const":
			k = completionKindConstant
		}
		items = append(items, completionItem{name, name, k, pkgPath + "." + name})
	}
	return items
}

func (j *jsEditor) findPackageDir(pkgPath string) string {
	// Check GOROOT
	goroot := os.Getenv("GOROOT")
	if goroot != "" {
		dir := filepath.Join(goroot, "src", pkgPath)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	// Check GOMODCACHE
	gmc := os.Getenv("GOMODCACHE")
	if gmc != "" {
		// For standard library-like paths (e.g., "github.com/foo/bar"), need to find in module cache
		dir := filepath.Join(gmc, pkgPath)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
		// Try to find in module cache with version suffix
		// e.g., /home/me/.cache/go-mod/github.com/foo/bar@v1.0.0/
		prefix := filepath.Join(gmc, pkgPath)
		parent := filepath.Dir(prefix)
		if info, err := os.Stat(parent); err == nil && info.IsDir() {
			entries, err := os.ReadDir(parent)
			if err == nil {
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), filepath.Base(prefix)+"@") {
						return filepath.Join(parent, entry.Name())
					}
				}
			}
		}
	}
	return ""
}

func (j *jsEditor) localDotCompletion(f *ast.File, dotPrefix string) []completionItem {
	// Simple local dot completion for struct fields
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == dotPrefix {
						st, ok := s.Type.(*ast.StructType)
						if ok && st.Fields != nil {
							var items []completionItem
							for _, field := range st.Fields.List {
								for _, name := range field.Names {
									if name.IsExported() {
										items = append(items, completionItem{
											name.Name, name.Name, completionKindField,
											typeExprString(field.Type),
										})
									}
								}
							}
							return items
						}
					}
				}
			}
		}
	}
	return nil
}

// wellKnownExports returns common exports for packages we can't parse from disk
func wellKnownExports(pkgPath string) []completionItem {
	known := map[string][]completionItem{
		"fmt": {
			{"Printf", "Printf", completionKindFunction, "func Printf(format string, a ...any) (n int, err error)"},
			{"Println", "Println", completionKindFunction, "func Println(a ...any) (n int, err error)"},
			{"Print", "Print", completionKindFunction, "func Print(a ...any) (n int, err error)"},
			{"Sprintf", "Sprintf", completionKindFunction, "func Sprintf(format string, a ...any) string"},
			{"Fprintf", "Fprintf", completionKindFunction, "func Fprintf(w io.Writer, format string, a ...any) (n int, err error)"},
			{"Errorf", "Errorf", completionKindFunction, "func Errorf(format string, a ...any) error"},
		},
		"os": {
			{"Open", "Open", completionKindFunction, "func Open(name string) (*File, error)"},
			{"Create", "Create", completionKindFunction, "func Create(name string) (*File, error)"},
			{"ReadFile", "ReadFile", completionKindFunction, "func ReadFile(name string) ([]byte, error)"},
			{"WriteFile", "WriteFile", completionKindFunction, "func WriteFile(name string, data []byte, perm FileMode) error"},
			{"Stdout", "Stdout", completionKindVariable, "*File"},
			{"Stderr", "Stderr", completionKindVariable, "*File"},
			{"Stdin", "Stdin", completionKindVariable, "*File"},
			{"Args", "Args", completionKindVariable, "[]string"},
			{"Exit", "Exit", completionKindFunction, "func Exit(code int)"},
			{"Getenv", "Getenv", completionKindFunction, "func Getenv(key string) string"},
			{"MkdirAll", "MkdirAll", completionKindFunction, "func MkdirAll(path string, perm FileMode) error"},
		},
		"strings": {
			{"Join", "Join", completionKindFunction, "func Join(elems []string, sep string) string"},
			{"Split", "Split", completionKindFunction, "func Split(s, sep string) []string"},
			{"Contains", "Contains", completionKindFunction, "func Contains(s, substr string) bool"},
			{"HasPrefix", "HasPrefix", completionKindFunction, "func HasPrefix(s, prefix string) bool"},
			{"HasSuffix", "HasSuffix", completionKindFunction, "func HasSuffix(s, suffix string) bool"},
			{"TrimSpace", "TrimSpace", completionKindFunction, "func TrimSpace(s string) string"},
			{"ToLower", "ToLower", completionKindFunction, "func ToLower(s string) string"},
			{"ToUpper", "ToUpper", completionKindFunction, "func ToUpper(s string) string"},
			{"Replace", "Replace", completionKindFunction, "func Replace(s, old, new string, n int) string"},
			{"Builder", "Builder", completionKindClass, "struct"},
		},
		"time": {
			{"Now", "Now", completionKindFunction, "func Now() Time"},
			{"Sleep", "Sleep", completionKindFunction, "func Sleep(d Duration)"},
			{"Since", "Since", completionKindFunction, "func Since(t Time) Duration"},
			{"Duration", "Duration", completionKindClass, "type Duration int64"},
			{"Second", "Second", completionKindConstant, "Duration"},
			{"Millisecond", "Millisecond", completionKindConstant, "Duration"},
		},
		"encoding/json": {
			{"Marshal", "Marshal", completionKindFunction, "func Marshal(v any) ([]byte, error)"},
			{"Unmarshal", "Unmarshal", completionKindFunction, "func Unmarshal(data []byte, v any) error"},
			{"NewEncoder", "NewEncoder", completionKindFunction, "func NewEncoder(w io.Writer) *Encoder"},
			{"NewDecoder", "NewDecoder", completionKindFunction, "func NewDecoder(r io.Reader) *Decoder"},
		},
		"net/http": {
			{"Get", "Get", completionKindFunction, "func Get(url string) (resp *Response, err error)"},
			{"Post", "Post", completionKindFunction, "func Post(url, contentType string, body io.Reader) (resp *Response, err error)"},
			{"ListenAndServe", "ListenAndServe", completionKindFunction, "func ListenAndServe(addr string, handler Handler) error"},
			{"HandleFunc", "HandleFunc", completionKindFunction, "func HandleFunc(pattern string, handler func(ResponseWriter, *Request))"},
			{"ServeMux", "ServeMux", completionKindClass, "struct"},
			{"ResponseWriter", "ResponseWriter", completionKindClass, "interface"},
			{"Request", "Request", completionKindClass, "struct"},
		},
	}
	return known[pkgPath]
}
