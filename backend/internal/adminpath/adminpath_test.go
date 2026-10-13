package adminpath

import "testing"

func TestNormalizeRejectsAdminAndReserved(t *testing.T) {
	if _, msg := Normalize("admin"); msg == "" {
		t.Fatal("admin was accepted")
	}
	if _, msg := Normalize("api"); msg == "" {
		t.Fatal("api was accepted")
	}
	if _, msg := Normalize("Ab"); msg == "" {
		t.Fatal("short path was accepted")
	}
	got, msg := Normalize("/Desk9/")
	if msg != "" || got != "desk9" {
		t.Fatalf("normalize = %q %q", got, msg)
	}
	path := Random()
	if _, msg := Normalize(path); msg != "" || path == "admin" {
		t.Fatalf("random = %q", path)
	}
}
