package calibre


type ListItem interface {
	ID() uint64
	Name() string
}



type listItem struct {
	id uint64
	name string
}

func (li *listItem) Name() string { return li.name }
func (li *listItem) ID() uint64 { return li.id }
