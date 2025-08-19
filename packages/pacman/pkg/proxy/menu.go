package proxy

import "github.com/caseymrm/menuet"

type StaticMenu []menuet.MenuItem

func (m StaticMenu) Children() []menuet.MenuItem {
	return m
}
