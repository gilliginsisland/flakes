package proxy

import "github.com/caseymrm/menuet"

// MenuNode embeds a pointer to a menuet.MenuItem and manages its children.
type MenuNode struct {
	*menuet.MenuItem
	nodes    []*MenuNode
	children []menuet.MenuItem
	Refresh  func()
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
	if n.Refresh != nil {
		n.Refresh()
	}
	return n.children
}
