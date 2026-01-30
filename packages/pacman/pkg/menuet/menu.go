package menuet

type Menuer interface {
	MenuItems() []Itemer
}

type Section struct {
	Title   string
	Content Menuer
}

func (s *Section) MenuItems() []Itemer {
	var children []Itemer
	if s.Content != nil {
		children = s.Content.MenuItems()
	}
	items := make([]Itemer, 1+len(children))
	items[0] = &MenuItemSectionHeader{
		Text: s.Title,
	}
	copy(items[1:], children)
	return items
}

type Sections []Menuer

func (ss Sections) MenuItems() []Itemer {
	var children [][]Itemer
	if n := len(ss); n == 0 {
		return nil
	} else {
		children = make([][]Itemer, n)
	}

	var total int
	for i, m := range ss {
		items := m.MenuItems()
		children[i] = items
		total += len(items) + 1 // +1 for separator
	}

	out := make([]Itemer, total)
	idx := 0
	for _, items := range children {
		copy(out[idx:], items)
		idx += len(items)
		out[idx] = &MenuItemSeparator{}
		idx++
	}
	// remove last separator
	out = out[:total-1]

	return out
}

type StaticItem MenuItem

func (item *StaticItem) MenuItems() []Itemer {
	return []Itemer{
		(*MenuItem)(item),
	}
}

type StaticItems []Itemer

func (s StaticItems) MenuItems() []Itemer {
	return s
}

type DynamicItem func() Itemer

func (f DynamicItem) MenuItems() []Itemer {
	return []Itemer{f()}
}

type DynamicItems func() []Itemer

func (f DynamicItems) MenuItems() []Itemer {
	return f()
}
