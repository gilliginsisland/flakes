package app

import (
	"slices"

	"github.com/caseymrm/menuet"
)

// Menu owns the flat slice of MenuItems and manages groups.
type Menu struct {
	items  []menuet.MenuItem
	groups []*MenuGroup
}

// AddGroup creates an empty group at the end.
func (m *Menu) AddGroup() *MenuGroup {
	g := &MenuGroup{menu: m, start: len(m.items)}
	m.groups = append(m.groups, g)
	if len(m.groups) > 1 {
		g.AddChild(menuet.MenuItem{
			Type: menuet.Separator,
		})
	}
	return g
}

// insert handles inserting a MenuItem into the flat slice
// and rebinds all groups and nodes.
func (m *Menu) insert(g *MenuGroup, item menuet.MenuItem) *MenuNode {
	// grow flat slice
	abs := g.start + len(g.nodes)
	m.items = slices.Insert(m.items, abs, item)

	node := MenuNode{MenuItem: &m.items[abs]}
	g.nodes = append(g.nodes, &node)

	// shift groups
	for i := slices.Index(m.groups, g) + 1; i < len(m.groups); i++ {
		g := m.groups[i]
		g.start++
		for j := 0; j < len(g.nodes); j++ {
			g.nodes[j].MenuItem = &m.items[j+g.start]
		}
	}

	return &node
}

// Children returns the children slice for this node.
func (m *Menu) Children() []menuet.MenuItem {
	return m.items
}

// MenuGroup is a view into a contiguous portion of Menu.items.
type MenuGroup struct {
	menu  *Menu
	start int
	nodes []*MenuNode
}

// AddChild inserts a new child into this group and returns a MenuNode.
func (g *MenuGroup) AddChild(item menuet.MenuItem) *MenuNode {
	return g.menu.insert(g, item)
}

// MenuNode embeds a pointer to a menuet.MenuItem and manages its children.
type MenuNode struct {
	*menuet.MenuItem
	nodes    []*MenuNode
	children []menuet.MenuItem
}

// AddChild creates a child MenuNode from a menuet.MenuItem.
func (n *MenuNode) AddChild(item menuet.MenuItem) *MenuNode {
	var node MenuNode

	// append to the children slice
	n.children = append(n.children, item)
	n.nodes = append(n.nodes, &node)

	// ensure all pointers are rebound to the current backing array
	for i := range len(n.children) {
		n.nodes[i].MenuItem = &n.children[i]
	}

	// ensure this node’s Children is wired if it now has at least one child
	if n.MenuItem != nil {
		n.MenuItem.Children = n.Children
	}

	return &node
}

// Children returns the children slice for this node.
func (n *MenuNode) Children() []menuet.MenuItem {
	return n.children
}
