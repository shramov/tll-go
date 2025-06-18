package tll

import "testing"

func assertEqual[T comparable](t *testing.T, a, b T) {
	if a != b {
		t.Errorf("Assertion failed: %v != %v", a, b)
	}
}

func TestConfig(t *testing.T) {
	cfg := NewConfig()
	defer cfg.Free()
	cfg.Set("a.b", "0")
	cfg.Set("a.c", "1")
	cfg.Set("a.d", "2")
	assertEqual(t, cfg.Get("a.x"), nil)
	assertEqual(t, *cfg.Get("a.b"), "0")
	r := cfg.Browse("a.*")
	defer r.Free()
	assertEqual(t, len(r.List), 3)
	assertEqual(t, r.List[0].Key, "a.b")
	assertEqual(t, r.List[1].Key, "a.c")
	assertEqual(t, r.List[2].Key, "a.d")
	assertEqual(t, *r.List[0].Cfg.Value(), "0")
	assertEqual(t, *r.List[1].Cfg.Value(), "1")
	assertEqual(t, *r.List[2].Cfg.Value(), "2")
}
