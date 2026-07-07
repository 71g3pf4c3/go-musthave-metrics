package pool

import "testing"

type testStruct struct {
	Name  string
	Count int
	Items []string
}

func (t *testStruct) Reset() {
	t.Name = ""
	t.Count = 0
	t.Items = t.Items[:0]
}

func TestPool_GetPut(t *testing.T) {
	p := New(func() *testStruct {
		return &testStruct{}
	})

	obj := p.Get()
	obj.Name = "hello"
	obj.Count = 42
	obj.Items = append(obj.Items, "a", "b")

	p.Put(obj)

	obj2 := p.Get()
	if obj2.Name != "" {
		t.Errorf("expected empty Name after reset, got %q", obj2.Name)
	}
	if obj2.Count != 0 {
		t.Errorf("expected zero Count after reset, got %d", obj2.Count)
	}
	if len(obj2.Items) != 0 {
		t.Errorf("expected empty Items after reset, got %v", obj2.Items)
	}
}

func TestPool_NewCreatesObject(t *testing.T) {
	called := 0
	p := New(func() *testStruct {
		called++
		return &testStruct{}
	})

	obj := p.Get()
	if obj == nil {
		t.Fatal("expected non-nil object from Get")
	}
	if called != 1 {
		t.Errorf("expected constructor called once, got %d", called)
	}
}
