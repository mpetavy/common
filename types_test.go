package common

import "testing"

func TestContainsWildcard(t *testing.T) {
	b := ContainsWildcard("a?b")
	if !b {
		t.Fail()
	}

	b = ContainsWildcard("a*b")
	if !b {
		t.Fail()
	}

	b = ContainsWildcard("ab")
	if b {
		t.Fail()
	}
}

func TestEqualWildcards(t *testing.T) {
	b,err := EqualWildcards("test.go","test.go")
	if !b || err != nil {
		t.Fail()
	}

	b,err = EqualWildcards("test.go","test.goo")
	if b || err != nil {
		t.Fail()
	}

	b,err = EqualWildcards("test.go","*.go")
	if !b || err != nil {
		t.Fail()
	}

	b,err = EqualWildcards("test.go","test.*")
	if !b || err != nil {
		t.Fail()
	}

	b,err = EqualWildcards("test.go","??st.go")
	if !b || err != nil {
		t.Fail()
	}

	b,err = EqualWildcards("test.go","test.??")
	if !b || err != nil {
		t.Fail()
	}
}
