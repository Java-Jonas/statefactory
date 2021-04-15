package enginefactory

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"bar-cli/ast"

	"github.com/dave/jennifer/jen"
	"github.com/gertd/go-pluralize"
)

// TODO wtf
const isProductionMode = false

func title(name string) string {
	return strings.Title(name)
}

// pluralizeClient is used to find the singular of field names
// this is necessary for writing coherent method names, eg. in write_adders.go (toSingular)
// with getting the singular form of a plural, this field:
// { pieces []piece }
// can have the coherent adder method of "AddPiece"
var pluralizeClient *pluralize.Client = pluralize.NewClient()

type EngineFactory struct {
	config *ast.AST
	buf    *bytes.Buffer
}

func onlyIf(is bool, statement *jen.Statement) *jen.Statement {
	if is {
		return statement
	}
	return jen.Empty()
}

// WriteEngineFrom writes source code for a given State-/ActionsConfig
func WriteEngineFrom(stateConfigData map[interface{}]interface{}) []byte {
	config := ast.Parse(stateConfigData, map[interface{}]interface{}{})
	s := newStateFactory(config).
		writePackageName().
		writeOperationKind().
		writeIDs().
		writeState().
		writeEngine().
		writeGenerateID().
		writeUpdateState().
		writeElements().
		writeAdders().
		writeCreators().
		writeDeleters().
		writeGetters().
		writeRemovers().
		writeSetters().
		writeTree().
		writeTreeElements().
		writeAssembleTree().
		writeAssembleTreeElement().
		writeDeduplicate()

	err := s.format()
	if err != nil {
		// unexpected error
		panic(err)
	}

	return s.writtenSourceCode()
}

func (s *EngineFactory) writePackageName() *EngineFactory {
	s.buf.WriteString("package state\n")
	return s
}

func newStateFactory(config *ast.AST) *EngineFactory {
	return &EngineFactory{
		config: config,
		buf:    &bytes.Buffer{},
	}
}

func (s *EngineFactory) writtenSourceCode() []byte {
	return s.buf.Bytes()
}

func (s *EngineFactory) format() error {
	config, err := parser.ParseFile(token.NewFileSet(), "", s.buf.String(), parser.AllErrors)
	if err != nil {
		return err
	}

	s.buf.Reset()
	err = format.Node(s.buf, token.NewFileSet(), config)
	return err
}