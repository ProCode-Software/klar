package klarml

import "errors"

var (
	SkipRest    = errors.New("skip the rest of the parent object")
	StopWalk    = errors.New("stop the walk")
	SkipCurrent = errors.New("skip this node")
)

type VisitFunc = func(*Node) (Node, error)

func walkAll[T Node](nodes *[]T, fn VisitFunc) error {
loop:
	for i := range *nodes {
		err := walk(&(*nodes)[i], fn)
		switch err {
		case nil:
		case StopWalk:
			return err
		case SkipCurrent:
			continue loop
		case SkipRest:
			return SkipRest
		}
	}
	return nil
}

func walk[T Node](node *T, fn VisitFunc) (err error) {
	n := Node(*node)
	n, err = fn(&n)
	*node = n.(T)
	if err != nil {
		return err
	}
	switch n := n.(type) {
	case Object:
		walkAll(&n.Properties, fn)
		*node = Node(n).(T)
	case Array:
		walkAll(&n.Items, fn)
		*node = Node(n).(T)
	case Document:
		err = walkAll(&n.Variables, fn)
		*node = Node(n).(T)
		if err != nil {
			return err
		}
		err = walk(&n.Body, fn)
		*node = Node(n).(T)
		if err != nil {
			return err
		}
		walkAll(&n.Comments, fn)
		*node = Node(n).(T)
	case Property:
		err = walk(&n.Value, fn)
		*node = Node(n).(T)
	case VarDecl:
		err = walk(&n.Value, fn)
		*node = Node(n).(T)
	}
	return err
}

func Walk[T Node](node T, fn VisitFunc) (T, error) {
	err := walk(&node, fn)
	switch err {
	case nil, SkipRest, SkipCurrent, StopWalk:
		return node, nil
	default:
		return node, err
	}
}
