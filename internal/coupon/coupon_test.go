package coupon

import "testing"

func TestValidator_Valid_length(t *testing.T) {
	v := &Validator{}     // zero sets: no code matches two files
	if v.Valid("SHORT") { // 5 chars
		t.Fatal("expected invalid length")
	}
	if v.Valid("1234567") { // 7 chars
		t.Fatal("expected invalid length")
	}
	if v.Valid("12345678901") { // 11 chars
		t.Fatal("expected invalid length")
	}
}

func TestValidator_Valid_requiresTwoFiles(t *testing.T) {
	key := makeKey([]byte("TESTCODE")) // 8 chars
	set := map[tokenKey]struct{}{key: {}}
	v := &Validator{
		sets: [3]map[tokenKey]struct{}{
			set,
			set,
			nil,
		},
	}
	if !v.Valid("TESTCODE") {
		t.Fatal("expected valid when present in two files")
	}
	v2 := &Validator{
		sets: [3]map[tokenKey]struct{}{
			set,
			nil,
			nil,
		},
	}
	if v2.Valid("TESTCODE") {
		t.Fatal("expected invalid when only in one file")
	}
}
