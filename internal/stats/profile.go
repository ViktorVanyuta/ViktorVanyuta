// Package stats описывает сводные данные профиля GitHub и их агрегацию.
//
// Пакет выделен отдельно, чтобы сборщик данных (internal/githubapi) и
// отрисовка карточек (internal/render) зависели от одного типа, а не друг от
// друга: иначе render пришлось бы импортировать клиент API вместе с его
// сетевым кодом.
package stats

import "sort"

// Profile — сводные данные по аккаунту за всё время. Приватные репозитории
// учитываются по счётчикам, исходный код наружу не отдаётся.
type Profile struct {
	Login         string
	ReposPublic   int
	ReposPrivate  int
	Languages     map[string]int
	CommitsByYear map[string]int
	Source        string
}

// TotalCommits — сумма коммитов по всем годам.
func (p *Profile) TotalCommits() int {
	total := 0
	for _, count := range p.CommitsByYear {
		total += count
	}
	return total
}

// TotalBytes — суммарный объём языков в байтах.
func (p *Profile) TotalBytes() int {
	total := 0
	for _, size := range p.Languages {
		total += size
	}
	return total
}

// ReposTotal — публичные и приватные репозитории вместе.
func (p *Profile) ReposTotal() int { return p.ReposPublic + p.ReposPrivate }

// LanguageShare — доля языка по объёму кода.
type LanguageShare struct {
	Name  string
	Bytes int
}

// Share — процент языка от общего объёма.
func (l LanguageShare) Share(total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(l.Bytes) / float64(total) * 100
}

// RankedLanguages — языки по убыванию объёма, при равенстве — по алфавиту,
// чтобы карточка не менялась от запуска к запуску.
func (p *Profile) RankedLanguages() []LanguageShare {
	shares := make([]LanguageShare, 0, len(p.Languages))
	for name, size := range p.Languages {
		shares = append(shares, LanguageShare{Name: name, Bytes: size})
	}
	sort.Slice(shares, func(i, j int) bool {
		if shares[i].Bytes != shares[j].Bytes {
			return shares[i].Bytes > shares[j].Bytes
		}
		return shares[i].Name < shares[j].Name
	})
	return shares
}
