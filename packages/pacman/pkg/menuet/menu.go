package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

#import "menuet.h"

*/
import "C"

type Section struct {
	Title   string
	Content Itemer
}

func (s *Section) item() *C.MenuItem {
	return toMenuItems([]Itemer{
		&MenuItemSectionHeader{
			Text: s.Title,
		},
		s.Content,
	})
}

type Sections []Itemer

func (ss Sections) item() *C.MenuItem {
	var children []Itemer
	n := len(ss)
	if n == 0 {
		return nil
	}
	children = make([]Itemer, n*2)

	for i, m := range ss {
		children[i*2] = m
		children[i*2+1] = &MenuItemSeparator{}
	}

	return toMenuItems(children[0 : n*2-1])
}

type StaticItem = MenuItem

type StaticItems []Itemer

func (s StaticItems) item() *C.MenuItem {
	return toMenuItems(s)
}

type DynamicItem func() Itemer

func (f DynamicItem) item() *C.MenuItem {
	return f().item()
}

type DynamicItems func() []Itemer

func (f DynamicItems) item() *C.MenuItem {
	return toMenuItems(f())
}
