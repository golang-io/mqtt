package topic

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type node struct {
	path string // 路由过滤器的部分
	// pattern string //
	m    sync.RWMutex
	next map[string]*node
}

func (n *node) print(m int, w io.Writer) {
	paths := n.paths()
	fmt.Fprintf(w, "%spath=%s, next=%#v\n", strings.Repeat("\t", m), n.path, paths)
	for _, path := range paths {
		n.next[path].print(m+1, w)
	}
}

func newNode(path string) *node {
	return &node{path: path, next: make(map[string]*node)}
}

// Add node
func (n *node) add(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	n.m.Lock()
	defer n.m.Unlock()
	current := n
	for _, subPath := range strings.Split(path, "/") {
		if _, ok := current.next[subPath]; !ok {
			current.next[subPath] = newNode(subPath)
		}
		current = current.next[subPath]
	}
	return nil
}

func (n *node) remove(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if _, ok := n.find(path); !ok { // 如果不是订阅的主题不允许删除!
		return fmt.Errorf("topic not found")
	}

	current := n
	for _, subPath := range strings.Split(path, "/") {
		if next, ok := current.get(subPath); ok {

			if len(next.next) == 0 {
				current.m.Lock()
				delete(current.next, subPath)
				current.m.Unlock()
			}
			current = next
		}
	}
	return nil
}

func (n *node) get(path string) (*node, bool) {
	n.m.RLock()
	defer n.m.RUnlock()
	next, ok := n.next[path]
	return next, ok
}

func (n *node) find(path string) ([]string, bool) {
	current := n
	var subs []string
	for _, p := range strings.Split(path, "/") {
		if next, ok := current.get("#"); ok {
			subs = append(subs, next.path)
			return subs, true
		}
		next, ok := current.get(p)
		if !ok {
			if next, ok = current.get("+"); !ok {
				return subs, false
			}
		}
		subs = append(subs, next.path)
		current = next
	}
	return subs, true
}

func (n *node) paths() []string {
	var v []string
	for k := range n.next {
		v = append(v, k)
	}
	return v
}

func (n *node) Print(w io.Writer) {
	n.print(0, w)
}

type MemoryTrie struct {
	root *node // 主题过滤树
}

func NewMemoryTrie() *MemoryTrie {
	return &MemoryTrie{
		root: newNode(""),
	}
}

func (m *MemoryTrie) Print(w io.Writer) {
	m.root.Print(w)
}

// Subscribe 订阅
func (m *MemoryTrie) Subscribe(topicName string) error {
	return m.root.add(topicName)
}

func (m *MemoryTrie) Unsubscribe(topicName string) {
	m.root.remove(topicName)
}

func (m *MemoryTrie) Find(topicName string) ([]string, bool) {
	return m.root.find(topicName)
}
