package leaks

import (
	"strings"
	"testing"
)

func TestScanFindsTokens(t *testing.T) {
	cases := []string{
		"token=ghp_" + strings.Repeat("a", 36),
		"github_pat_" + strings.Repeat("B", 22) + "_" + strings.Repeat("c", 25),
		"gho_" + strings.Repeat("1", 30),
		"ghs_" + strings.Repeat("z", 30),
		"ghs_" + strings.Repeat("z", 30),
		"gho_" + strings.Repeat("0", 40),
	}
	for _, payload := range cases {
		if findings := Scan("test", payload); len(findings) == 0 {
			t.Errorf("не найден токен в %q", payload[:12]+"...")
		}
	}
}

func TestScanIgnoresShortAndEmbedded(t *testing.T) {
	cases := []string{
		"ghp_" + strings.Repeat("a", 20),
		"xghp_" + strings.Repeat("a", 36),
		"github_pat_" + strings.Repeat("a", 20),
		"просто текст без токенов",
	}
	for _, payload := range cases {
		if findings := Scan("test", payload); len(findings) != 0 {
			t.Errorf("ложное срабатывание на %q: %v", payload, findings)
		}
	}
}

func TestIsBinary(t *testing.T) {
	if !isBinary("assets/logo.PNG") {
		t.Error("PNG должен считаться бинарным")
	}
	if isBinary("main.go") {
		t.Error("main.go не бинарный")
	}
}

func TestErrLeaksFoundMessage(t *testing.T) {
	if ErrLeaksFound.Error() == "" {
		t.Error("ErrLeaksFound должен иметь текст")
	}
}
