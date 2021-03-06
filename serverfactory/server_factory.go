package serverfactory

import (
	"bytes"

	. "github.com/jobergner/backent-cli/factoryutils"

	"github.com/jobergner/backent-cli/ast"
)

type ServerFactory struct {
	config *ast.AST
	buf    *bytes.Buffer
}

// isIDTypeOfType evaluates whether a given type name is the respective ID-Type
// of a user-defined type.
// Background:
// Every user-defined type has a generated ID type.
// E.g. a defined type "person" has its ID-Type "PersonID" generated automatically
func (s ServerFactory) isIDTypeOfType(typeName string) bool {
	for _, configType := range s.config.Types {
		if configType.Name+"ID" == typeName {
			return true
		}
	}
	return false
}

func newServerFactory(config *ast.AST) *ServerFactory {
	return &ServerFactory{
		config: config,
		buf:    &bytes.Buffer{},
	}
}

// WriteServerFrom writes source code for a given ActionsConfig
func WriteServer(
	buf *bytes.Buffer,
	stateConfigData, actionsConfigData, responsesConfigData map[interface{}]interface{},
	configJson []byte,
) {
	config := ast.Parse(stateConfigData, actionsConfigData, responsesConfigData)
	s := newServerFactory(config).
		writePackageName(). // to be able to format the code without errors
		writeMessageKinds().
		writeParameters().
		writeResponses().
		writeProcessClientMessage().
		writeInspectHandler(configJson)

	err := Format(s.buf)
	if err != nil {
		// unexpected error
		panic(err)
	}

	// TODO: comments were being swallowed during format. find out why
	s.writeActions().
		writeSideEffects()

	buf.WriteString(TrimPackageName(s.buf.String()))
}

func (s *ServerFactory) writePackageName() *ServerFactory {
	s.buf.WriteString("package state\n")
	return s
}
