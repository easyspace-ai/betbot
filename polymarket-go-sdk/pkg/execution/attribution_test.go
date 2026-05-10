package execution

import "testing"

func TestNormalizeAttribution(t *testing.T) {
	got := NormalizeAttribution(Attribution{
		Builder: " builder-a ",
		Funder:  " 0xabc ",
		Source:  " PMX-Gateway ",
	})
	if got.Builder != "builder-a" {
		t.Fatalf("unexpected builder: %q", got.Builder)
	}
	if got.Funder != "0xabc" {
		t.Fatalf("unexpected funder: %q", got.Funder)
	}
	if got.Source != "pmx-gateway" {
		t.Fatalf("unexpected source: %q", got.Source)
	}
}

func TestAttributionHeaderMap(t *testing.T) {
	a := Attribution{
		Builder: "builder-a",
		Funder:  "0xabc",
		Source:  "gateway",
	}
	m := a.HeaderMap()
	if m[HeaderAttributionBuilder] != "builder-a" {
		t.Fatalf("missing builder header")
	}
	if m[HeaderAttributionFunder] != "0xabc" {
		t.Fatalf("missing funder header")
	}
	if m[HeaderAttributionSource] != "gateway" {
		t.Fatalf("missing source header")
	}
}
