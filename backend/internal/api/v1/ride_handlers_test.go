package v1

import "testing"

func TestPriceCents(t *testing.T) {
	for raw, want := range map[string]int64{"0": 0, "0.01": 1, "20.5": 2050, "20.25": 2025, "999999.99": 99999999} {
		got, ok := priceCents(raw)
		if !ok || got != want {
			t.Errorf("invalid cents conversion for %q", raw)
		}
	}
	for _, raw := range []string{"", "-1", "1000000", "1.001", "NaN", "1e2", "01", ".5", "1.", " 1"} {
		if _, ok := priceCents(raw); ok {
			t.Errorf("accepted invalid price %q", raw)
		}
	}
}
