// +build unit

package utils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTagToNumber(t *testing.T) {
	cases := []struct {
		tag      string
		val      int
		Expected bool
	}{
		{"v0.0.1", 1, true},
		{"v0.1.1", 1001, true},
		{"v1.1.1", 1001001, true},
		{"v0.0.10", 10, true},
		{"v0.10.10", 10010, true},
		{"v10.10.10", 10010010, true},
		{"v0.5.0", 5000, true},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s", tc.tag), func(t *testing.T) {
			if tc.Expected {
				assert.Equal(t, TagToNumber(tc.tag), tc.val)
			}
		})
	}
}

func TestIsTagDeployable(t *testing.T) {
	tags := []string{"v0.0.1", "v0.0.2", "v0.1.0", "v1.0.0", "random-branch-name"}
	cases := []struct {
		tag      string
		tags     []string
		Expected bool
	}{
		{"v0.0.1", tags, true},
		{"v0.5.0", tags, false},
		{"random-branch-name", tags, true},
		{"", tags, true},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s in %s", tc.tag, tc.tags), func(t *testing.T) {
			assert.Equal(t, IsTagDeployable(tc.tag, tc.tags), tc.Expected)
		})
	}
}

func TestIsTagReleaseFormat(t *testing.T) {
	cases := []struct {
		tag      string
		Expected bool
	}{
		{"a0.0.1", false},
		{"v0.0.1", true},
		{"v0.0.11", true},
		{"v0.0.111", true},
		{"v0.0.11a", false},
		{"v0.0.1111", false},
		{"v999.999.999", true},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s, %t", tc.tag, tc.Expected), func(t *testing.T) {
			assert.Equal(t, IsTagReleaseFormat(tc.tag), tc.Expected)
		})
	}
}
