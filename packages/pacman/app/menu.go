package app

import (
	"net"

	"github.com/caseymrm/menuet"
	"github.com/gilliginsisland/pacman/pkg/iterutil"
)

type Menuer interface {
	MenuItems() []menuet.MenuItem
}

type Section struct {
	Title   string
	Content Menuer
}

func (s Section) MenuItems() []menuet.MenuItem {
	var children []menuet.MenuItem
	if s.Content != nil {
		children = s.Content.MenuItems()
	}
	items := make([]menuet.MenuItem, 1+len(children))
	items[0] = menuet.MenuItem{
		Text:       s.Title,
		FontWeight: menuet.WeightMedium,
	}
	copy(items[1:], children)
	return items
}

type Sections []Menuer

func (ss Sections) MenuItems() []menuet.MenuItem {
	var children [][]menuet.MenuItem
	if n := len(ss); n == 0 {
		return nil
	} else {
		children = make([][]menuet.MenuItem, n)
	}

	var total int
	for i, m := range ss {
		items := m.MenuItems()
		children[i] = items
		total += len(items) + 1 // +1 for separator
	}

	out := make([]menuet.MenuItem, total)
	idx := 0
	for _, items := range children {
		copy(out[idx:], items)
		idx += len(items)
		out[idx] = menuet.MenuItem{Type: menuet.Separator}
		idx++
	}
	// remove last separator
	out = out[:total-1]

	return out
}

type StaticItems []menuet.MenuItem

func (s StaticItems) MenuItems() []menuet.MenuItem {
	return s
}

type StaticItem menuet.MenuItem

func (item StaticItem) MenuItems() []menuet.MenuItem {
	return []menuet.MenuItem{
		menuet.MenuItem(item),
	}
}

type AddrItem struct {
	Listener net.Listener
}

func (ai *AddrItem) MenuItems() []menuet.MenuItem {
	return []menuet.MenuItem{{
		Text:       ai.Listener.Addr().String(),
		FontWeight: menuet.WeightLight,
	}}
}

func (dp DialerPool) MenuItems() []menuet.MenuItem {
	items := make([]menuet.MenuItem, len(dp))
	var idx int
	for _, pd := range iterutil.SortedMapIter(dp) {
		items[idx] = pd.MenuItem()
		idx++
	}
	return items
}
