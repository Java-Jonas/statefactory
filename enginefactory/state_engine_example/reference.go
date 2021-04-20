package state

type itemBoundToRef struct {
	id       PlayerID
	parentID ItemID
}

func (ref itemBoundToRef) IsSet(se *Engine) bool {
	return ref.id != 0
}

func (ref itemBoundToRef) Unset(se *Engine) {
	ref.id = 0
	item := se.Item(ref.parentID).item
	item.BoundTo = ref
	se.updateItem(item)
}

func (ref itemBoundToRef) Set(se *Engine, id PlayerID) {
	ref.id = id
	item := se.Item(ref.parentID).item
	item.BoundTo = ref
	se.updateItem(item)
}

func (ref itemBoundToRef) Get(se *Engine) player {
	return se.Player(ref.id)
}

type playerGuildMembersSliceRef struct {
	id       PlayerID
	parentID PlayerID
}

func (ref playerGuildMembersSliceRef) Get(se *Engine) player {
	return se.Player(ref.id)
}