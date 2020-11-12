package statefactory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddSetters(t *testing.T) {
	t.Run("adds setters", func(t *testing.T) {
		input := unsafeParseDecls([]string{
			output_person_type,
			output_name_type,
			output_child_type,
		})

		actual := splitPrintedDeclarations(input.addSetters())
		expected := []string{
			output_person_type,
			output_name_type,
			output_child_type,
			output_SetAge_person_func,
			output_SetFirst_name_func,
			output_SetLast_name_func,
		}

		missingDeclarations, redundantDeclarations := matchDeclarations(actual, expected)

		assert.Equal(t, []string{}, missingDeclarations)
		assert.Equal(t, []string{}, redundantDeclarations)
	})
}

func (sm *stateMachine) addSetters() *stateMachine {
	return sm
}
