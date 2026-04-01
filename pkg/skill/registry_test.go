package skill_test

import (
	"sync"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadSkill(t *testing.T, name string) skill.Skill {
	t.Helper()
	content := "---\nname: " + name + "\ndescription: " + name + " skill\nversion: \"1.0\"\n---\nBody of " + name + "."
	s, err := skill.LoadFromString(content)
	require.NoError(t, err)
	return s
}

func TestNewRegistry(t *testing.T) {
	r := skill.NewRegistry()
	assert.NotNil(t, r)
	assert.Empty(t, r.All())
	assert.Empty(t, r.Names())
}

func TestRegister_Success(t *testing.T) {
	r := skill.NewRegistry()
	err := r.Register(loadSkill(t, "foo"))
	require.NoError(t, err)

	s, ok := r.Get("foo")
	assert.True(t, ok)
	assert.Equal(t, "foo", s.Header().Name)
}

func TestRegister_DuplicateError(t *testing.T) {
	r := skill.NewRegistry()
	require.NoError(t, r.Register(loadSkill(t, "dup")))

	err := r.Register(loadSkill(t, "dup"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestGet_Found(t *testing.T) {
	r := skill.NewRegistry()
	require.NoError(t, r.Register(loadSkill(t, "target")))

	s, ok := r.Get("target")
	assert.True(t, ok)
	assert.Equal(t, "target", s.Header().Name)
}

func TestGet_NotFound(t *testing.T) {
	r := skill.NewRegistry()
	s, ok := r.Get("missing")
	assert.False(t, ok)
	assert.Nil(t, s)
}

func TestAll_Sorted(t *testing.T) {
	r := skill.NewRegistry()
	require.NoError(t, r.Register(loadSkill(t, "charlie")))
	require.NoError(t, r.Register(loadSkill(t, "alpha")))
	require.NoError(t, r.Register(loadSkill(t, "bravo")))

	all := r.All()
	require.Len(t, all, 3)
	assert.Equal(t, "alpha", all[0].Header().Name)
	assert.Equal(t, "bravo", all[1].Header().Name)
	assert.Equal(t, "charlie", all[2].Header().Name)
}

func TestNames_Sorted(t *testing.T) {
	r := skill.NewRegistry()
	require.NoError(t, r.Register(loadSkill(t, "zulu")))
	require.NoError(t, r.Register(loadSkill(t, "alpha")))
	require.NoError(t, r.Register(loadSkill(t, "mike")))

	names := r.Names()
	assert.Equal(t, []string{"alpha", "mike", "zulu"}, names)
}

func TestRemove(t *testing.T) {
	r := skill.NewRegistry()
	require.NoError(t, r.Register(loadSkill(t, "removeme")))

	r.Remove("removeme")
	_, ok := r.Get("removeme")
	assert.False(t, ok)
	assert.Empty(t, r.Names())
}

func TestRemove_NonexistentIsNoOp(t *testing.T) {
	r := skill.NewRegistry()
	// Should not panic or error.
	r.Remove("ghost")
	assert.Empty(t, r.Names())
}

func TestConcurrentAccess(t *testing.T) {
	r := skill.NewRegistry()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent registrations with unique names.
	for i := range goroutines {
		name := "skill-" + string(rune('A'+i))
		s := loadSkill(t, name)
		go func() {
			defer wg.Done()
			_ = r.Register(s)
		}()
	}

	// Concurrent reads.
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = r.Names()
		}()
	}

	// Concurrent gets.
	for i := range goroutines {
		name := "skill-" + string(rune('A'+i))
		go func() {
			defer wg.Done()
			r.Get(name)
		}()
	}

	wg.Wait()

	// Verify the registry is still consistent.
	all := r.All()
	names := r.Names()
	assert.Equal(t, len(all), len(names))
}
