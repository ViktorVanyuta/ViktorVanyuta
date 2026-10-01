package render

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/ViktorVanyuta/ViktorVanyuta/internal/stats"
)

func TestFormatNumber(t *testing.T) {
	cases := map[int]string{0: "0", 7: "7", 214: "214", 1000: "1 000", 1234567: "1 234 567"}
	for input, want := range cases {
		if got := formatNumber(input); got != want {
			t.Errorf("formatNumber(%d) = %q, ожидалось %q", input, got, want)
		}
	}
}

func TestDonutPathFullCircle(t *testing.T) {
	// Сектор в 100% обязан состоять из двух дуг, иначе он вырождается в точку.
	path := donutPath(100, 100, 80, 50, 0, tau)
	if arcs := strings.Count(path, "A"); arcs != 4 {
		t.Errorf("в полном круге %d дуг, ожидалось 4: %s", arcs, path)
	}
}

func TestDonutPathHalfCircle(t *testing.T) {
	path := donutPath(100, 100, 80, 50, 0, math.Pi)
	if arcs := strings.Count(path, "A"); arcs != 2 {
		t.Errorf("в полукруге %d дуг, ожидалось 2: %s", arcs, path)
	}
	if !strings.HasPrefix(path, "M180.00,100.00") {
		t.Errorf("неожиданное начало пути: %s", path)
	}
}

func TestOverview(t *testing.T) {
	profile := &stats.Profile{
		Login:         "ViktorVanyuta",
		ReposPublic:   1,
		ReposPrivate:  25,
		CommitsByYear: map[string]int{"2022": 5, "2026": 9},
		Source:        "GraphQL contributionsCollection",
	}
	svg := Overview(profile)

	for _, want := range []string{
		"<svg", "</svg>", "ViktorVanyuta", "14", "26",
		"публичные: 1", "приватные: 25", "GraphQL contributionsCollection",
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("в overview.svg нет %q:\n%s", want, svg)
		}
	}
	if strings.Contains(svg, "1 4") {
		t.Errorf("числа переформатированы неверно:\n%s", svg)
	}
	if strings.Contains(svg, "Период активности") {
		t.Errorf("период активности не должен попадать в карточку:\n%s", svg)
	}
}

func TestLanguages(t *testing.T) {
	profile := &stats.Profile{Languages: map[string]int{"Go": 75, "Kotlin": 25}}
	svg := Languages(profile)

	for _, want := range []string{
		"<path", "Go", "Kotlin", "75.0%", "25.0%", "2", "языков",
		"включая приватные",
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("в languages.svg нет %q:\n%s", want, svg)
		}
	}
}

func TestLanguagesWithoutData(t *testing.T) {
	svg := Languages(&stats.Profile{})
	if !strings.Contains(svg, "нет данных о языках") {
		t.Errorf("пустой профиль должен давать заглушку:\n%s", svg)
	}
	if strings.Contains(svg, "<path") {
		t.Errorf("пустой профиль не должен рисовать секторы:\n%s", svg)
	}
}

// svgHeight читает высоту карточки из готового SVG.
func svgHeight(t *testing.T, svg string) float64 {
	t.Helper()
	match := regexp.MustCompile(`<svg[^>]*height="([\d.]+)"`).FindStringSubmatch(svg)
	if match == nil {
		t.Fatalf("в SVG нет атрибута height:\n%s", svg)
	}
	height, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		t.Fatalf("высота %q не число: %v", match[1], err)
	}
	return height
}

// TestLegendDoesNotOverlapFootnote — регрессия: при девяти языках легенда
// наезжала на подпись «по всем репозиториям».
func TestLegendDoesNotOverlapFootnote(t *testing.T) {
	languages := map[string]int{}
	for index, name := range []string{
		"Go", "Kotlin", "Python", "JavaScript", "HTML",
		"PLpgSQL", "Dockerfile", "CSS", "Shell",
	} {
		languages[name] = 1000 - index*100
	}

	svg := Languages(&stats.Profile{Languages: languages})
	height := svgHeight(t, svg)

	var rows []float64
	for _, match := range regexp.MustCompile(`<text x="246.0" y="([\d.]+)"`).FindAllStringSubmatch(svg, -1) {
		y, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			t.Fatalf("y %q не число: %v", match[1], err)
		}
		rows = append(rows, y)
	}
	if len(rows) != len(languages) {
		t.Fatalf("в легенде %d строк, ожидалось %d", len(rows), len(languages))
	}

	footnote := regexp.MustCompile(`<text x="24.0" y="([\d.]+)"[^>]*>по всем репозиториям`).FindStringSubmatch(svg)
	if footnote == nil {
		t.Fatalf("нет подписи о выборке:\n%s", svg)
	}
	footnoteY, _ := strconv.ParseFloat(footnote[1], 64)

	last := rows[len(rows)-1]
	if last >= footnoteY {
		t.Errorf("последняя строка легенды y=%.1f наезжает на подпись y=%.1f", last, footnoteY)
	}
	if footnoteY > height-8 {
		t.Errorf("подпись y=%.1f выходит за карточку высотой %.1f", footnoteY, height)
	}
}

// TestLanguagesCardGrowsWithManyLanguages — при большом числе языков карточка
// увеличивается вместо того, чтобы налезать легендой на подпись.
func TestLanguagesCardGrowsWithManyLanguages(t *testing.T) {
	languages := map[string]int{}
	for index := 0; index < 20; index++ {
		languages[fmt.Sprintf("Lang%d", index)] = 1000 - index*10
	}

	svg := Languages(&stats.Profile{Languages: languages})
	height := svgHeight(t, svg)
	if height <= cardHeight {
		t.Errorf("при 20 языках высота осталась %.0f, ожидался рост карточки", height)
	}

	rows := regexp.MustCompile(`<text x="246.0" y="([\d.]+)"`).FindAllStringSubmatch(svg, -1)
	last, _ := strconv.ParseFloat(rows[len(rows)-1][1], 64)
	if last >= height-8 {
		t.Errorf("последняя строка легенды y=%.1f выходит за карточку высотой %.1f", last, height)
	}
}

// TestCardsShareSize — обе карточки стоят рядом в одной таблице README, поэтому
// их базовая высота должна совпадать.
func TestCardsShareSize(t *testing.T) {
	profile := &stats.Profile{
		Login:         "b",
		ReposPublic:   1,
		ReposPrivate:  25,
		Languages:     map[string]int{"Go": 75, "Kotlin": 25},
		CommitsByYear: map[string]int{"2024": 3},
		Source:        "GraphQL contributionsCollection",
	}

	overviewHeight := svgHeight(t, Overview(profile))
	languagesHeight := svgHeight(t, Languages(profile))

	if overviewHeight != languagesHeight {
		t.Errorf("высоты карточек разные: overview %.0f, languages %.0f", overviewHeight, languagesHeight)
	}
	if overviewHeight != cardHeight {
		t.Errorf("высота карточки %.0f, ожидалась базовая %.0f", overviewHeight, cardHeight)
	}
}

// TestOverviewHasNoActivityPeriod — период активности убран из карточки по
// просьбе владельца профиля.
func TestOverviewHasNoActivityPeriod(t *testing.T) {
	svg := Overview(&stats.Profile{
		Login:         "b",
		ReposPublic:   1,
		ReposPrivate:  25,
		CommitsByYear: map[string]int{"2022": 16, "2026": 68},
	})

	for _, unwanted := range []string{"Период активности", "2022", "2026"} {
		if strings.Contains(svg, unwanted) {
			t.Errorf("в overview.svg не должно быть %q:\n%s", unwanted, svg)
		}
	}
}
