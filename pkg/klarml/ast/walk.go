package ast

import "errors"

var (
	SkipRest    = errors.New("skip the rest of the parent object")
	StopWalk    = errors.New("stop the walk")
	SkipCurrent = errors.New("skip this node")
)

type VisitFunc = func(Node) error

func walkAll[T Node](nodes []T, fn VisitFunc) error {
loop:
	for i := range nodes {
		err := walk(nodes[i], fn)
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

func walk(node Node, fn VisitFunc) (err error) {
	err = fn(node)
	if err != nil {
		return err
	}
	switch n := node.(type) {
	case *Object:
		walkAll(n.Properties, fn)
	case *Array:
		walkAll(n.Items, fn)
	case *Document:
		err = walkAll(n.Variables, fn)
		if err != nil {
			return err
		}
		err = walk(n.Body, fn)
		if err != nil {
			return err
		}
		walkAll(n.Comments, fn)
	case *Property:
		err = walk(n.Value, fn)
	case *VarDecl:
		err = walk(n.Value, fn)
	}
	return err
}

func Walk(node Node, fn VisitFunc) error {
	err := walk(node, fn)
	switch err {
	case nil, SkipRest, SkipCurrent, StopWalk:
		return nil
	default:
		return err
	}
}
