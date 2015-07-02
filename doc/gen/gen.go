package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"euphoria.io/heim/proto"
)

type objects doc.Package

func (o *objects) tmplObject(name string) (command, error) {
	for _, typ := range o.Types {
		if typ.Name == name {
			cmd := command{
				Name: name,
				Doc:  typ.Doc,
			}

			if len(typ.Decl.Specs) > 0 {
				if ts, ok := typ.Decl.Specs[0].(*ast.TypeSpec); ok {
					cmd.Fields = getFields(o, ts.Type)
				}
			}

			return cmd, nil
		}
	}

	return command{}, fmt.Errorf("object not found: %s", name)
}

type packets map[string]command

func (ps packets) tmplPacket(name string) (command, error) {
	if cmd, ok := ps[name]; ok {
		cmd.Displayed = true
		ps[name] = cmd
		return cmd, nil
	}
	return command{}, fmt.Errorf("packet not found: %s", name)
}

func (ps packets) tmplOthers() []command {
	commands := []command{}
	for _, cmd := range ps {
		if !cmd.Displayed && cmd.Name != "PresenceEvent" {
			commands = append(commands, cmd)
		}
	}
	sort.Sort(commandList(commands))
	return commands
}

type types map[string]string

func (types) link(name string) string { return strings.ToLower(name) }

func (t types) registerType(name string) string {
	t[name] = t.link(name)
	return ""
}

func (t types) linkType(name string) string {
	switch {
	case strings.HasPrefix(name, "[]"):
		return fmt.Sprintf("[%s]", t.linkType(name[2:]))
	case name == "Listing":
		return t.linkType("[]SessionView")
	case name == "snowflake.Snowflake":
		return t.linkType("Snowflake")
	case name == "json.RawMessage":
		return "object"
	default:
		if link, ok := t[name]; ok {
			return fmt.Sprintf("[%s](#%s)", name, link)
		}
		fmt.Fprintf(os.Stderr, "undocumented field type: %s\n", name)
		return fmt.Sprintf("`%s`", name)
	}
}

type command struct {
	Name      string
	Doc       string
	Fields    []field
	Displayed bool
}

type commandList []command

func (cl commandList) Len() int           { return len(cl) }
func (cl commandList) Less(i, j int) bool { return cl[i].Name < cl[j].Name }
func (cl commandList) Swap(i, j int)      { cl[i], cl[j] = cl[j], cl[i] }

type field struct {
	Name     string
	TypeName string
	Optional bool
	Comments string
}

func sortObjects(obs *objects) packets {
	ps := packets{}
	for commandName, typeName := range proto.PacketsByType() {
		cmd, err := obs.tmplObject(typeName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: resolve error: %s\n", err)
			continue
		}
		ps[string(commandName)] = cmd
	}
	return ps
}

func getFields(obs *objects, node ast.Node) []field {
	switch typ := node.(type) {
	case *ast.StructType:
		fields := make([]field, 0, len(typ.Fields.List))
		for _, f := range typ.Fields.List {
			if len(f.Names) == 0 {
				// embedded type
				expr := f.Type
				for {
					if subexpr, ok := expr.(*ast.StarExpr); ok {
						expr = subexpr.X
						continue
					}
					break
				}
				switch ft := expr.(type) {
				case *ast.Ident:
					fields = append(fields, getFields(obs, ft)...)
				default:
					fmt.Fprintf(os.Stderr, "skipping field: %#v\n", f)
				}
				continue
			}
			nf := field{
				TypeName: typeIdent(f.Type),
				Comments: joinComments(f.Comment),
			}
			nf.Name, nf.Optional = nameAndOptional(f)
			fields = append(fields, nf)
		}
		return fields
	case *ast.Ident:
		for _, t := range obs.Types {
			if t.Name == typ.Name {
				if len(t.Decl.Specs) > 0 {
					if ts, ok := t.Decl.Specs[0].(*ast.TypeSpec); ok {
						return getFields(obs, ts.Type)
					}
				}
			}
		}
		return nil
	default:
		return nil
	}
}

func typeIdent(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", typeIdent(t.X), t.Sel.Name)
	case *ast.StarExpr:
		return typeIdent(t.X)
	case *ast.ArrayType:
		return fmt.Sprintf("[]%s", typeIdent(t.Elt))
	default:
		return fmt.Sprintf("%#v", expr)
	}
}

func nameAndOptional(f *ast.Field) (string, bool) {
	name := f.Names[0].Name
	optional := false

	if fieldTag, err := strconv.Unquote(f.Tag.Value); err == nil {
		jsonTag := reflect.StructTag(fieldTag).Get("json")
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			name = parts[0]
		}
		for _, part := range parts[1:] {
			if part == "omitempty" {
				optional = true
				break
			}
		}
	}

	return name, optional
}

func joinComments(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	b := &bytes.Buffer{}
	for _, c := range cg.List {
		text := c.Text
		if strings.HasPrefix(text, "//") {
			text = text[2:]
		}
		b.WriteString(text)
	}
	return b.String()
}

func run() error {
	pkg, err := build.Import("euphoria.io/heim/proto", "", build.FindOnly)
	if err != nil {
		return fmt.Errorf("import error: %s", err)
	}

	if pkg.SrcRoot == "" {
		return fmt.Errorf("error: can't find source for package euphoria.io/heim/proto")
	}

	pkgs, err := parser.ParseDir(
		token.NewFileSet(), filepath.Join(pkg.SrcRoot, "euphoria.io/heim/proto"), nil,
		parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse error: %s", err)
	}

	obs := (*objects)(doc.New(pkgs["proto"], "euphoria.io/heim/proto", 0))
	ps := sortObjects(obs)
	ts := types{}
	t := template.New("api.md").Funcs(template.FuncMap{
		"object": obs.tmplObject,
		"others": ps.tmplOthers,
		"packet": ps.tmplPacket,

		"linkType":     ts.linkType,
		"registerType": ts.registerType,
	})

	ts.registerType("bool")
	ts.registerType("int")
	ts.registerType("object")
	ts.registerType("string")
	ts.registerType("AuthOption")
	ts.registerType("Message")
	ts.registerType("PacketType")
	ts.registerType("SessionView")
	ts.registerType("Snowflake")
	ts.registerType("Time")
	ts.registerType("UserID")

	gendir := filepath.Join(pkg.SrcRoot, "euphoria.io/heim/doc/gen")
	if err := os.Chdir(gendir); err != nil {
		return fmt.Errorf("chdir error: %s: %s", gendir, err)
	}

	if _, err := t.ParseGlob("*.md"); err != nil {
		return fmt.Errorf("template parse error: %s", err)
	}
	if err := t.Execute(os.Stdout, nil); err != nil {
		return fmt.Errorf("template render error: %s", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
