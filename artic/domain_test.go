package artic

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "artic" {
		t.Errorf("Scheme = %q, want artic", info.Scheme)
	}
	if info.Identity.Binary != "artic" {
		t.Errorf("Identity.Binary = %q, want artic", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("249839")
	if err != nil {
		t.Fatal(err)
	}
	if typ != "artwork" {
		t.Errorf("type = %q, want artwork", typ)
	}
	if id != "249839" {
		t.Errorf("id = %q, want 249839", id)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("artwork", "249839")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://www.artic.edu/artworks/249839"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestLocateUnknown(t *testing.T) {
	_, err := Domain{}.Locate("artist", "249839")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}
