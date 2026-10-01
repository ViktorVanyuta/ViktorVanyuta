// Package leaks ищет в репозитории строки, похожие на токены GitHub.
//
// Проверяются не только файлы, но и вся история коммитов: токен, попавший в
// репозиторий однажды, считается утечкой навсегда.
package leaks

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ErrLeaksFound — команда check-leaks нашла что-то похожее на токен.
var ErrLeaksFound = errors.New("в репозитории найдены строки, похожие на токены GitHub")

// tokenPatterns — сигнатуры токенов GitHub. Проверяется и история коммитов:
// токен, попавший в репозиторий однажды, считается утечкой навсегда.
var tokenPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"GitHub PAT (classic)", regexp.MustCompile(`\bghp_[A-Za-z0-9]{30,}`)},
	{"GitHub fine-grained PAT", regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{40,}`)},
	{"GitHub OAuth token", regexp.MustCompile(`\bgho_[A-Za-z0-9]{30,}`)},
	{"GitHub app token", regexp.MustCompile(`\bgh[us]_[A-Za-z0-9]{30,}`)},
	{"GitHub refresh token", regexp.MustCompile(`\bghr_[A-Za-z0-9]{30,}`)},
	{"GitHub server-to-server", regexp.MustCompile(`\bghs_[A-Za-z0-9]{30,}`)},
}

// binarySuffixes — файлы, которые нечитаемы как текст и пропускаются.
var binarySuffixes = []string{
	".png", ".jpg", ".jpeg", ".gif", ".ico", ".pdf", ".zip", ".gz",
	".woff", ".woff2", ".ttf", ".mp4", ".webm", ".DS_Store",
}

// Finding — одно совпадение сигнатуры токена.
type Finding struct {
	Label   string
	Kind    string
	Preview string
	Offset  int
}

// Scan ищет сигнатуры токенов в тексте.
func Scan(label, payload string) []Finding {
	var findings []Finding
	for _, candidate := range tokenPatterns {
		for _, match := range candidate.re.FindAllStringIndex(payload, -1) {
			start, end := match[0], match[1]
			preview := payload[start:min(start+8, end)]
			findings = append(findings, Finding{
				Label:   label,
				Kind:    candidate.name,
				Preview: preview,
				Offset:  start,
			})
		}
	}
	return findings
}

func isBinary(path string) bool {
	lower := strings.ToLower(path)
	for _, suffix := range binarySuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// ScanFiles читает текстовые файлы из списка и ищет в них токены.
func ScanFiles(paths []string) []Finding {
	var findings []Finding
	for _, path := range paths {
		if path == "" || isBinary(path) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		findings = append(findings, Scan(path, string(data))...)
	}
	return findings
}

// ReportFindings печатает результат и возвращает true, если что-то найдено.
func ReportFindings(findings []Finding) bool {
	if len(findings) == 0 {
		return false
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Label != findings[j].Label {
			return findings[i].Label < findings[j].Label
		}
		return findings[i].Offset < findings[j].Offset
	})

	fmt.Fprintln(os.Stderr, "ОБНАРУЖЕНЫ ПОХОЖИЕ НА ТОКЕНЫ:")
	seen := make(map[string]bool, len(findings))
	for _, finding := range findings {
		line := fmt.Sprintf("  %s: %s (начинается с %s…, позиция %d)",
			finding.Label, finding.Kind, finding.Preview, finding.Offset)
		if seen[line] {
			continue
		}
		seen[line] = true
		fmt.Fprintln(os.Stderr, line)
	}
	fmt.Fprint(os.Stderr,
		"\nЕсли это настоящий токен — отзови его на GitHub немедленно:\n"+
			"  Settings -> Developer settings -> Personal access tokens -> Revoke\n")
	return true
}
