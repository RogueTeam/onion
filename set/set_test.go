package set_test

import (
	"testing"

	"github.com/RogueTeam/onion/set"
	"github.com/stretchr/testify/assert"
)

func TestRawSet_String(t *testing.T) {
	assertions := assert.New(t)

	var s = set.New(1, 2, 3)

	assertions.Contains(s.String(), "{")
	assertions.Contains(s.String(), "1")
	assertions.Contains(s.String(), "2")
	assertions.Contains(s.String(), "3")
	assertions.Contains(s.String(), "}")
}

func TestRawSet_Clear(t *testing.T) {
	assertions := assert.New(t)

	var s = set.New(1, 2, 3)
	s.Clear()

	assertions.Empty(s)
}

func TestRawSet_Slice(t *testing.T) {
	assertions := assert.New(t)

	var s = set.New[string]("value")

	assertions.NotEmpty(s.Slice())
	assertions.Contains(s.Slice(), "value")
}

func TestRawSet_Add(t *testing.T) {
	assertions := assert.New(t)

	var s = set.New[string]()

	s.Add("value")

	assertions.NotEmpty(s)
	assertions.Contains(s, "value")
}

func TestRawSet_AddSet(t *testing.T) {
	assertions := assert.New(t)

	var s1 = set.New[string]()

	s1.Add("value")

	var s2 = set.New[string]()

	s2.AddSet(s1)

	assertions.NotEmpty(s2)
	assertions.Contains(s2, "value")
}

func TestRawSet_Del(t *testing.T) {
	assertions := assert.New(t)

	var s = set.New("value")

	s.Del("value")

	assertions.Empty(s)
	assertions.NotContains(s, "value")
}

func TestRawSet_Union(t *testing.T) {
	assertions := assert.New(t)

	var s1 = set.New("value1")

	var s2 = set.New("value2")

	s3 := s1.Union(s2)

	assertions.NotEmpty(s3)
	assertions.Contains(s3, "value1")
	assertions.Contains(s3, "value2")
}

func TestRawSet_Has(t *testing.T) {
	assertions := assert.New(t)

	var s = set.New("value")

	assertions.True(s.Has("value"))
}

func TestRawSet_Intersection(t *testing.T) {
	assertions := assert.New(t)

	var (
		s1 = set.New("value1")
		s2 = set.New("value1", "value2")
		s3 = s1.Intersection(s2)
	)

	assertions.Contains(s3, "value1")
	assertions.NotContains(s3, "value2")
}

func TestRawSet_Difference(t *testing.T) {
	assertions := assert.New(t)

	var (
		s1 = set.New("value1", "value2")
		s2 = set.New("value1", "value3")
		s3 = s1.Difference(s2)
	)

	assertions.Contains(s3, "value2")
	assertions.NotContains(s3, "value1")
	assertions.NotContains(s3, "value3")
}

func TestRawSet_Subset(t *testing.T) {
	t.Run("Is Subset", func(t *testing.T) {
		assertions := assert.New(t)

		var (
			s1 = set.New("value1", "value2", "value3", "value4")
			s2 = set.New("value1", "value2", "value3", "value4", "value5", "value6", "value7")
		)

		assertions.True(s1.Subset(s2))
	})
	t.Run("Is not Subset", func(t *testing.T) {
		assertions := assert.New(t)

		var (
			s1 = set.New("INVALID")
			s2 = set.New("value1", "value2", "value3", "value4", "value5", "value6", "value7")
		)

		assertions.False(s1.Subset(s2))
	})
}
