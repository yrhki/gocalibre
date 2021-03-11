package calibre



type Author struct {
	id uint64
	name string
}

func (a *Author) Name() string { return a.name }
func (a *Author) ID() uint64 { return a.id }


