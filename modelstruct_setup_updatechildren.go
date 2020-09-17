package aorm

func (this *ModelStruct) updateChildren() {
	if len(this.Children) == 0 {
		return
	}

	for _, child := range this.ChildrenByName {
		child.Parent = this
		for _, childField := range child.FieldsByName {
			childField.BaseModel = child
			if childField.IsChild {
				childField.Relationship.Model = child
			} else if childField.Relationship != nil && childField.Relationship != nil {
				if childField.Relationship.AssociationModel.Type == child.Type {
					childField.Relationship.AssociationModel = child
				}
				if childField.Relationship.JoinTableHandler != nil {
					jth := childField.Relationship.JoinTableHandler.(*JoinTableHandler)
					jth.source.ModelStruct = child
				}
			}
		}
		child.ParentField.Relationship.AssociationModel = child
		child.updateChildren()
	}
}
