// Code generated by "stringer -type=repoProvider"; DO NOT EDIT.

package repository

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ProviderUnknown-0]
	_ = x[ProviderS3-1]
}

const _repoProvider_name = "ProviderUnknownProviderS3"

var _repoProvider_index = [...]uint8{0, 15, 25}

func (i repoProvider) String() string {
	if i < 0 || i >= repoProvider(len(_repoProvider_index)-1) {
		return "repoProvider(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _repoProvider_name[_repoProvider_index[i]:_repoProvider_index[i+1]]
}