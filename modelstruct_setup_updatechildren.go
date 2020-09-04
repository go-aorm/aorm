package aorm

func (this *ModelStruct) updateChildren() {
	if len(this.Children) == 0 {
		return
	}

	for _, child := range this.ChildrenByName {
		child.Parent = this
		for _, childField := range child.FieldsByName {
			if childField.IsChild {
				childField.Relationship.Model = child
			}
		}
		child.ParentField.Relationship.AssociationModel = child
		child.updateChildren()
	}
}
