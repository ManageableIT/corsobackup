// Code generated by "stringer -type=optionIdentifier"; DO NOT EDIT.

package exchange

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[unknown-0]
	_ = x[folders-1]
	_ = x[messages-2]
	_ = x[users-3]
}

const _optionIdentifier_name = "unknownfoldersmessagesusers"

var _optionIdentifier_index = [...]uint8{0, 7, 14, 22, 27}

func (i optionIdentifier) String() string {
	if i < 0 || i >= optionIdentifier(len(_optionIdentifier_index)-1) {
		return "optionIdentifier(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _optionIdentifier_name[_optionIdentifier_index[i]:_optionIdentifier_index[i+1]]
}