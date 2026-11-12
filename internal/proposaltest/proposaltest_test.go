package proposaltest

import (
	"reflect"
	"sync"
	"testing"

	"github.com/progrium/taragen/internal/design"
)

func TestTags(t *testing.T) {
	for name, want := range map[string][]string{
		"renders [baseline]":        {"baseline"},
		"renders [a, b-c]":          {"a", "b-c"},
		"renders":                   nil,
		"renders [a,b]":             nil,
		"[a] in the middle renders": nil,
	} {
		if got := Tags(name); !reflect.DeepEqual(got, want) {
			t.Errorf("Tags(%q) = %v, want %v", name, got, want)
		}
	}
}

func useFixture(t *testing.T) {
	r, err := design.Load("testdata/repo")
	if err != nil {
		t.Fatal(err)
	}
	once.Do(func() {})
	repo, loadErr = r, nil
	t.Cleanup(func() { once, repo, loadErr = sync.Once{}, nil, nil })
}

func TestGate(t *testing.T) {
	useFixture(t)
	ran := map[string]bool{}
	for _, name := range []string{"untagged", "implemented [done]", "pending [later]", "mixed [done, later]"} {
		t.Run(name, func(t *testing.T) {
			Gate(t, Tags(name)...)
			ran[name] = true
		})
	}
	want := map[string]bool{"untagged": true, "implemented [done]": true}
	if !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}

	t.Setenv("DESIGN_PENDING", "1")
	pending := false
	Run(t, "pending [later]", func(t *testing.T) { pending = true })
	if !pending {
		t.Error("DESIGN_PENDING=1 didn't run a pending case")
	}
}
