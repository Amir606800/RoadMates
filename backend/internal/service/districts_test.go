package service

import "testing"

func TestFoldDistrict(t *testing.T) {
	for input, want := range map[string]string{
		" Şamaxı ": "samaxi", "ŞAMAXI": "samaxi", "Əə Iı İi Öö Üü Çç Şş Ğğ": "ee ii ii oo uu cc ss gg", "GÖYÇAY": "goycay", "  samaxi  ": "samaxi",
	} {
		if got := FoldDistrict(input); got != want {
			t.Errorf("FoldDistrict(%q)=%q, want %q", input, got, want)
		}
	}
}
