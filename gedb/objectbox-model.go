// Code generated by ObjectBox; DO NOT EDIT.

package gedb

import (
	"github.com/objectbox/objectbox-go/objectbox"
)

// ObjectBoxModel declares and builds the model from all the entities in the package.
// It is usually used when setting-up ObjectBox as an argument to the Builder.Model() function.
func ObjectBoxModel() *objectbox.Model {
	model := objectbox.NewModel()
	model.GeneratorVersion(5)

	model.RegisterBinding(GeDatasBinding)
	model.LastEntityId(1, 8326878006900731439)

	return model
}
