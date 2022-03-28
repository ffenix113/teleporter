package manager

import (
	"strings"
)

type Tree struct {
	File *File
	Tree TreeRoot
}

type TreeRoot map[string]*Tree

func (t Tree) IsFile() bool {
	return t.File != nil
}

func (t Tree) IsDir() bool {
	return t.Tree != nil
}

func (t Tree) FilesInfo() []*File {
	if t.IsFile() {
		return []*File{t.File}
	}

	infos := make([]*File, 0, len(t.Tree))
	for path, tree := range t.Tree {
		switch {
		case tree.IsFile():
			infos = append(infos, tree.File)
		default:
			infos = append(infos, &File{
				Path:  path,
				IsDir: true,
			})
		}
	}

	return infos
}

func (t *Tree) Add(path string, tree *Tree) {
	head := t

	parts := strings.Split(path, "/")
	for _, part := range parts[:len(parts)-1] {
		if part == "" {
			continue
		}

		if head.Tree == nil {
			head.Tree = TreeRoot{}
		}

		if _, ok := head.Tree[part]; !ok {
			head.Tree[part] = &Tree{}
		}

		head = head.Tree[part]
	}

	if head.Tree == nil {
		head.Tree = TreeRoot{}
	}

	head.Tree[parts[len(parts)-1]] = tree
}

func (t *Tree) Delete(path string) {
	head := t

	parts := strings.Split(path, "/")
	for _, part := range parts[:len(parts)-1] {
		if part == "" {
			continue
		}

		if head.Tree == nil {
			return
		}

		if _, ok := head.Tree[part]; !ok {
			return
		}

		head = head.Tree[part]
	}

	delete(head.Tree, parts[len(parts)-1])
}

type searchable interface {
	*File | *Tree
}

func FindInTree[T searchable](t *Tree, path string) (T, bool) {
	var res T
	found := t

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		parts = nil
	}

	for _, part := range parts {
		if found.Tree == nil {
			return res, false
		}

		found = found.Tree[part]
	}

	if found == nil {
		return res, false
	}

	var foundRes any
	switch any(res).(type) {
	case *File:
		if found.File == nil {
			return res, false
		}

		foundRes = found.File
	case *Tree:
		foundRes = found
	}

	return foundRes.(T), true
}
