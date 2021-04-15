package serverfactory

import (
	"bytes"
	"strings"

	"bar-cli/ast"
)

type ServerFactory struct {
	config *ast.AST
	buf    *bytes.Buffer
}

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

func title(name string) string {
	return strings.Title(name)
}