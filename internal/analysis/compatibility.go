package analysis

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) IsCompatibleType(expType, gotType Type) bool {
	switch expType := expType.(type) {
	case types.Union:
		return slices.Contains(expType.Options, gotType)
	case types.List:
		
	}
	return false
}

func (c *Checker) ToTyped(typ, hint Type) (Type, errors.KlarError) {
	return nil, nil
}

func (c *Checker) CheckCompatible(
	expected Type, expr Expression, ctx context,
) (gotType Type, ok bool) {
	//gotType = c._typeWithHint(expr, expected, ctx)
	if expected == gotType {
		return expected, true
	}
	return types.InvalidType, false
}
