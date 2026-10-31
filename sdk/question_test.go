package vocab

import (
	"testing"
)

func TestNormalizeSpellingAndProficiencyScore(t *testing.T) {
	if got := NormalizeSpellingGuess("  Naïve  "); got != "naive" {
		t.Errorf("NormalizeSpellingGuess('  Naïve  ') = %q, want 'naive'", got)
	}
	if score := ProficiencyScoreFromAbility(0.0); score != 1100 {
		t.Errorf("ProficiencyScoreFromAbility(0.0) = %d, want 1100", score)
	}
}
