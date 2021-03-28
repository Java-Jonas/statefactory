package enginefactory

import (
	. "github.com/dave/jennifer/jen"
)

func (s *stateFactory) writeCreators() *stateFactory {
	decls := newDeclSet()
	s.ast.rangeTypes(func(configType stateConfigType) {

		cw := creatorWrapper{
			t: configType,
		}

		decls.file.Func().Params(cw.receiverParams()).Id(cw.name()).Params().Id(cw.returns()).Block(
			Return(cw.createElement()),
		)

		c := creator{
			t: configType,
			f: nil,
		}

		decls.file.Func().Params(c.receiverParams()).Id(c.name()).Params(c.params()).Id(c.returns()).Block(
			c.declareElement(),
			c.generateID(),
			onlyIf(!configType.IsRootType, c.setHasParent()),
			forEachFieldInType(configType, func(field stateConfigField) *Statement {
				c.f = &field
				if field.HasSliceValue || field.ValueType.IsBasicType {
					return Empty()
				}
				return &Statement{
					c.createChildElement(), Line(),
					c.setChildElement(),
				}
			}),
			c.setOperationKind(),
			c.updateElementInPatch(),
			Return(c.returnElement()),
		)
	})

	decls.render(s.buf)
	return s
}

type creatorWrapper struct {
	t stateConfigType
}

func (cw creatorWrapper) receiverParams() *Statement {
	return Id("se").Id("*Engine")
}

func (cw creatorWrapper) name() string {
	return "Create" + title(cw.t.Name)
}

func (cw creatorWrapper) returns() string {
	return title(cw.t.Name)
}

func (cw creatorWrapper) createElement() *Statement {
	var callParam *Statement
	if cw.t.IsRootType {
		callParam = Empty()
	} else {
		callParam = Lit(false)
	}

	return Id("se").Dot("create" + title(cw.t.Name)).Call(callParam)
}

type creator struct {
	t stateConfigType
	f *stateConfigField
}

func (c creator) receiverParams() *Statement {
	return Id("se").Id("*Engine")
}

func (c creator) name() string {
	return "create" + title(c.t.Name)
}

func (c creator) returns() string {
	return title(c.t.Name)
}

func (c creator) hasParentParam() string {
	if c.t.IsRootType {
		return ""
	}
	return "hasParent"
}

func (c creator) params() *Statement {
	if c.t.IsRootType {
		return Empty()
	}
	return Id(c.hasParentParam()).Bool()
}

func (c creator) declareElement() *Statement {
	return Var().Id(c.t.Name).Id(c.t.Name + "Core")
}

func (c creator) generateID() *Statement {
	return Id(c.t.Name).Dot("ID").Op("=").Id(title(c.t.Name) + "ID").Call(Id("se").Dot("GenerateID").Call())
}

func (c creator) setHasParent() *Statement {
	return Id(c.t.Name).Dot("HasParent_").Op("=").Id(c.hasParentParam())
}

func (c creator) createChildElement() *Statement {
	return Id("element" + title(c.f.Name)).Op(":=").Id("se").Dot("create" + title(c.f.ValueType.Name)).Call(Lit(true))
}
func (c creator) setChildElement() *Statement {
	return Id(c.t.Name).Dot(title(c.f.Name)).Op("=").Id("element" + title(c.f.Name)).Dot(c.f.ValueType.Name).Dot("ID")
}

func (c creator) setOperationKind() *Statement {
	return Id(c.t.Name).Dot("OperationKind_").Op("=").Id("OperationKindUpdate")
}

func (c creator) updateElementInPatch() *Statement {
	return Id("se").Dot("Patch").Dot(title(c.t.Name)).Index(Id(c.t.Name).Dot("ID")).Op("=").Id(c.t.Name)
}

func (c creator) returnElement() *Statement {
	return Id(title(c.t.Name)).Values(Dict{
		Id(c.t.Name): Id(c.t.Name),
	})
}
