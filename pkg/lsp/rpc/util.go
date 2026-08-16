package rpc

import (
	"encoding/json/v2"
	"fmt"
)

type Union2[A, B any] struct{ Value any }

func (u *Union2[A, B]) IsNil() bool { return u.Value == nil }

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
