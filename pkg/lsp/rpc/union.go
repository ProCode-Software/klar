package rpc

import (
	"encoding/json/v2"
	"fmt"
)


type Union2[A, B any] struct{ Value any }

func (u *Union2[A, B]) IsNil() bool { return u == nil || u.Value == nil }

func (u *Union2[A, B]) Curr() int {
	switch u.Value.(type) {
	case nil:
		panic("union is nil")
	case A:
		return 0
	case B:
		return 1
	default:
		panic(fmt.Sprintf("unexpected type: %T", u.Value))
	}
}
func (u Union2[A, B]) A() A { return u.Value.(A) }
func (u Union2[A, B]) B() B { return u.Value.(B) }

func (u *Union2[A, B]) UnmarshalJSON(data []byte) (err error) {
	var a A
	if err = json.Unmarshal(data, &a); err == nil {
		u.Value = a
		return nil
	}
	var b B
	if err = json.Unmarshal(data, &b); err == nil {
		u.Value = b
		return nil
	}
	return err
}

func (u Union2[A, B]) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Value)
}

// 3-way
// ========

type Union3[A, B, C any] struct{ Value any }

func (u *Union3[A, B, C]) IsNil() bool { return u == nil || u.Value == nil }

func (u *Union3[A, B, C]) Curr() int {
	switch u.Value.(type) {
	case nil:
		panic("union is nil")
	case A:
		return 0
	case B:
		return 1
	case C:
		return 2
	default:
		panic(fmt.Sprintf("unexpected type: %T", u.Value))
	}
}
func (u Union3[A, B, C]) A() A { return u.Value.(A) }
func (u Union3[A, B, C]) B() B { return u.Value.(B) }
func (u Union3[A, B, C]) C() C { return u.Value.(C) }

func (u *Union3[A, B, C]) UnmarshalJSON(data []byte) (err error) {
	var a A
	if err = json.Unmarshal(data, &a); err == nil {
		u.Value = a
		return nil
	}
	var b B
	if err = json.Unmarshal(data, &b); err == nil {
		u.Value = b
		return nil
	}
	var c C
	if err = json.Unmarshal(data, &c); err == nil {
		u.Value = c
		return nil
	}
	return err
}

func (u Union3[A, B, C]) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Value)
}
