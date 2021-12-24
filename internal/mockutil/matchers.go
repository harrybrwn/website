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
