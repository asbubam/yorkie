package json

import (
	"strings"

	"github.com/hackerwins/yorkie/pkg/document/time"
	"github.com/hackerwins/yorkie/pkg/log"
)

type rgaNode struct {
	prev *rgaNode
	next *rgaNode
	elem Element
}

func (n *rgaNode) isDeleted() bool {
	return n.elem.DeletedAt() != nil
}

func newRGANode(elem Element) *rgaNode {
	return &rgaNode{
		prev: nil,
		next: nil,
		elem: elem,
	}
}

func newNodeAfter(prev *rgaNode, elem Element) *rgaNode {
	newNode := newRGANode(elem)
	prevNext := prev.next

	prev.next = newNode
	newNode.prev = prev
	newNode.next = prevNext
	if prevNext != nil {
		prevNext.prev = newNode
	}

	return prev.next
}

// RGA is replicated growable array.
type RGA struct {
	nodeMapByCreatedAt map[string]*rgaNode
	first              *rgaNode
	last               *rgaNode
	size               int
}

// NewRGA creates a new instance of RGA.
func NewRGA() *RGA {
	nodeMapByCreatedAt := make(map[string]*rgaNode)
	dummyHead := newRGANode(NewPrimitive("", time.InitialTicket))
	nodeMapByCreatedAt[dummyHead.elem.CreatedAt().Key()] = dummyHead

	return &RGA{
		nodeMapByCreatedAt: nodeMapByCreatedAt,
		first:              dummyHead,
		last:               dummyHead,
		size:               0,
	}
}

// Marshal returns the JSON encoding of this RGA.
func (a *RGA) Marshal() string {
	sb := strings.Builder{}
	sb.WriteString("[")

	idx := 0
	current := a.first.next
	for {
		if current == nil {
			break
		}

		if !current.isDeleted() {
			sb.WriteString(current.elem.Marshal())
			if a.size-1 != idx {
				sb.WriteString(",")
			}
			idx++
		}

		current = current.next
	}

	sb.WriteString("]")

	return sb.String()
}

// Add adds the given element at the last.
func (a *RGA) Add(elem Element) {
	a.insertAfter(a.last, elem)
}

// Nodes returns an array of elements contained in this RGA.
// TODO If we encounter performance issues, we need to replace this with other solution.
func (a *RGA) Nodes() []*rgaNode {
	var nodes []*rgaNode
	current := a.first.next
	for {
		if current == nil {
			break
		}
		nodes = append(nodes, current)
		current = current.next
	}

	return nodes
}

// LastCreatedAt returns the creation time of last elements.
func (a *RGA) LastCreatedAt() *time.Ticket {
	return a.last.elem.CreatedAt()
}

// InsertAfter inserts the given element after the given previous element.
func (a *RGA) InsertAfter(prevCreatedAt *time.Ticket, elem Element) {
	prevNode := a.findByCreatedAt(prevCreatedAt, elem.CreatedAt())
	a.insertAfter(prevNode, elem)
}

// Get returns the element of the given index.
func (a *RGA) Get(idx int) Element {
	// TODO introduce LLRBTree for improving upstream performance
	current := a.first.next
	for {
		if current == nil {
			break
		}
		if idx == 0 {
			return current.elem
		}

		current = current.next
		idx--
	}
	return nil
}

// RemoveByCreatedAt removes the given element.
func (a *RGA) RemoveByCreatedAt(createdAt *time.Ticket, deletedAt *time.Ticket) Element {
	if node, ok := a.nodeMapByCreatedAt[createdAt.Key()]; ok {
		node.elem.Delete(deletedAt)
		a.size--
		return node.elem
	}

	log.Logger.Warnf("fail to find the given createdAt: %s", createdAt.Key())
	return nil
}

// Len returns length of this RGA.
func (a *RGA) Len() int {
	return a.size
}

func (a *RGA) findByCreatedAt(prevCreatedAt *time.Ticket, createdAt *time.Ticket) *rgaNode {
	node := a.nodeMapByCreatedAt[prevCreatedAt.Key()]
	if node == nil {
		log.Logger.Fatalf(
			"fail to find the given prevCreatedAt: %s",
			prevCreatedAt.Key(),
		)
		return nil
	}

	for node.next != nil && createdAt.After(node.next.elem.CreatedAt()) {
		node = node.next
	}

	return node
}

func (a *RGA) insertAfter(prev *rgaNode, element Element) {
	newNode := newNodeAfter(prev, element)
	if prev == a.last {
		a.last = newNode
	}

	a.size++
	a.nodeMapByCreatedAt[element.CreatedAt().Key()] = newNode
}
