package statefactory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddRemovers(t *testing.T) {
	t.Run("adds removers", func(t *testing.T) {
		input := unsafeParseDecls([]string{
			output_person_type,
			output_name_type,
			output_child_type,
		})

		actual := splitPrintedDeclarations(input.addRemovers())
		expected := []string{
			output_person_type,
			output_name_type,
			output_child_type,
			output_RemovePerson_stateMachine_func,
			output_RemoveName_stateMachine_func,
			output_RemoveChild_stateMachine_func,
			output_RemoveChild_person_func,
		}

		missingDeclarations, redundantDeclarations := matchDeclarations(actual, expected)

		assert.Equal(t, []string{}, missingDeclarations)
		assert.Equal(t, []string{}, redundantDeclarations)
	})
}

func (sm *stateMachine) addRemovers() *stateMachine {
	return sm
}
