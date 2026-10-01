package stats

import "testing"

func TestLanguageShare(t *testing.T) {
	languages := map[string]int{"Go": 75, "Kotlin": 25}
	profile := &Profile{Languages: languages}

	if total := profile.TotalBytes(); total != 100 {
		t.Fatalf("TotalBytes() = %d, ожидалось 100", total)
	}
	ranked := profile.RankedLanguages()
	if len(ranked) != 2 || ranked[0].Name != "Go" || ranked[1].Name != "Kotlin" {
		t.Fatalf("RankedLanguages() = %v, ожидался Go раньше Kotlin", ranked)
	}
	if share := ranked[0].Share(profile.TotalBytes()); share != 75 {
		t.Errorf("доля Go = %v, ожидалось 75", share)
	}
	if share := (LanguageShare{Name: "X"}).Share(0); share != 0 {
		t.Errorf("доля X при нулевом total = %v, ожидалось 0", share)
	}
}

func TestTotals(t *testing.T) {
	profile := &Profile{
		ReposPublic:   1,
		ReposPrivate:  25,
		CommitsByYear: map[string]int{"2024": 10, "2025": 4},
	}
	if profile.ReposTotal() != 26 {
		t.Errorf("ReposTotal() = %d, ожидалось 26", profile.ReposTotal())
	}
	if profile.TotalCommits() != 14 {
		t.Errorf("TotalCommits() = %d, ожидалось 14", profile.TotalCommits())
	}
}

func TestRankedLanguagesIsStable(t *testing.T) {
	// При равных объёмах порядок не должен зависеть от порядка обхода карты.
	first := (&Profile{Languages: map[string]int{"Go": 10, "Kotlin": 10}}).RankedLanguages()
	second := (&Profile{Languages: map[string]int{"Kotlin": 10, "Go": 10}}).RankedLanguages()

	if first[0].Name != "Go" || second[0].Name != "Go" {
		t.Errorf("порядок нестабилен: %v и %v", first, second)
	}
}

func TestEmptyProfile(t *testing.T) {
	profile := &Profile{}
	if profile.TotalCommits() != 0 || profile.TotalBytes() != 0 || profile.ReposTotal() != 0 {
		t.Error("пустой профиль должен давать нули")
	}
	if len(profile.RankedLanguages()) != 0 {
		t.Error("у пустого профиля нет языков")
	}
}
