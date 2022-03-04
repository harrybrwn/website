package mockutil

import (
	"fmt"
	"strings"

	"github.com/golang/mock/gomock"
)

func HasPrefix(prefix string) gomock.Matcher {
	return &PrefixMatcher{prefix: prefix}
}

type PrefixMatcher struct {
	prefix string
}

func (pm *PrefixMatcher) Matches(x interface{}) bool {
	s, ok := x.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(s, pm.prefix)
}

func (pm *PrefixMatcher) String() string {
	return fmt.Sprintf("PrefixMatcher{%q}", pm.prefix)
}

func HasLen(n int) gomock.Matcher {
	return &LenMatcher{n: n}
}

type LenMatcher struct {
	n int
}

func (lm *LenMatcher) Matches(x interface{}) bool {
	var l int
	switch v := x.(type) {
	case string:
		l = len(v)
	case []byte:
		l = len(v)
	default:
		return false
	}
	return l == lm.n
}

func (lm *LenMatcher) String() string {
	return fmt.Sprintf("LenMatcher{%d}", lm.n)
}
