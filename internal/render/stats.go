package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ViktorVanyuta/ViktorVanyuta/internal/stats"
)

const tau = 2 * math.Pi

// Обе карточки одного размера: в README они стоят рядом.
const (
	cardWidth  = 520.0
	cardHeight = 268.0
)

// languageColors — цвета Linguist (подмножество языков, встречающихся в проекте).
var languageColors = map[string]string{
	"Go":          "#00ADD8",
	"JavaScript":  "#F1E05A",
	"TypeScript":  "#3178C6",
	"Python":      "#3572A5",
	"Java":        "#B07219",
	"Kotlin":      "#A97BFF",
	"Swift":       "#F05138",
	"Dart":        "#00B4AB",
	"C":           "#555555",
	"C++":         "#F34B7D",
	"C#":          "#178600",
	"HTML":        "#E34C26",
	"CSS":         "#563D7C",
	"SCSS":        "#C6538C",
	"Shell":       "#89E051",
	"Dockerfile":  "#384D54",
	"SQL":         "#E38C00",
	"YAML":        "#CB171E",
	"Makefile":    "#427819",
	"CMake":       "#DA3434",
	"PLpgSQL":     "#336790",
	"PHP":         "#4F5D95",
	"Rust":        "#DEA584",
	"Vue":         "#41B883",
	"Svelte":      "#FF3E00",
	"Objective-C": "#438EFF",
	"Groovy":      "#4298B8",
	"PowerShell":  "#012456",
	"Batchfile":   "#C1F12E",
	"Twig":        "#C1D026",
	"Blade":       "#F7523F",
	"HCL":         "#844FBA",
	"Nix":         "#7E7EFF",
}

// donutPath строит дугу кольцевого сектора. Полный круг рисуется двумя
// полукругами: иначе единственный сектор в 100% выродился бы в точку.
func donutPath(cx, cy, rOuter, rInner, start, end float64) string {
	if end-start >= tau-1e-9 {
		middle := start + math.Pi
		return donutPath(cx, cy, rOuter, rInner, start, middle) +
			donutPath(cx, cy, rOuter, rInner, middle, end)
	}

	large := 0
	if end-start > math.Pi {
		large = 1
	}

	outerStartX := cx + rOuter*math.Cos(start)
	outerStartY := cy + rOuter*math.Sin(start)
	outerEndX := cx + rOuter*math.Cos(end)
	outerEndY := cy + rOuter*math.Sin(end)
	innerEndX := cx + rInner*math.Cos(end)
	innerEndY := cy + rInner*math.Sin(end)
	innerStartX := cx + rInner*math.Cos(start)
	innerStartY := cy + rInner*math.Sin(start)

	return fmt.Sprintf(
		"M%.2f,%.2f A%.2f,%.2f 0 %d 1 %.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 %d 0 %.2f,%.2f Z",
		outerStartX, outerStartY, rOuter, rOuter, large, outerEndX, outerEndY,
		innerEndX, innerEndY, rInner, rInner, large, innerStartX, innerStartY)
}

// Overview — сводка: репозитории и коммиты за всё время.
func Overview(profile *stats.Profile) string {
	width, height := cardWidth, cardHeight

	parts := []string{
		svgOpen(width, height),
		card(width, height),
		txtBold(24, 40, profile.Login+" — всё время", 17, colorText),
	}

	tiles := []struct {
		label string
		value int
	}{
		{"Коммитов", profile.TotalCommits()},
		{"Репозиториев", profile.ReposTotal()},
	}
	for index, tile := range tiles {
		x := 24 + float64(index)*240
		parts = append(parts,
			txtBold(x, 96, formatNumber(tile.value), 40, colorGreen),
			txt(x, 118, tile.label, 12, colorMuted))
	}

	parts = append(parts, txt(24, 158, "Репозитории по доступности", 12, colorText))

	const barX, barY, barH = 24.0, 170.0, 14.0
	barW := width - 48
	if total := profile.ReposTotal(); total > 0 {
		publicWidth := float64(profile.ReposPublic) / float64(total) * barW
		parts = append(parts,
			fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.2f" height="%.1f" fill="%s"/>`,
				barX, barY, publicWidth, barH, colorBlue),
			fmt.Sprintf(`<rect x="%.2f" y="%.1f" width="%.2f" height="%.1f" rx="3" fill="%s"/>`,
				barX+publicWidth, barY, barW-publicWidth, barH, colorGreen))
	}

	legend := []struct {
		color string
		label string
		value int
	}{
		{colorBlue, "публичные", profile.ReposPublic},
		{colorGreen, "приватные", profile.ReposPrivate},
	}
	for index, item := range legend {
		x := 24 + float64(index)*200
		parts = append(parts,
			fmt.Sprintf(`<rect x="%.1f" y="200" width="11" height="11" rx="2" fill="%s"/>`, x, item.color),
			txt(x+19, 210, fmt.Sprintf("%s: %d", item.label, item.value), 12, colorText))
	}

	parts = append(parts,
		txt(24, height-22, "Источник: "+profile.Source, 10, colorMuted),
		"</svg>")

	return strings.Join(parts, "\n")
}

// Languages — кольцевая диаграмма используемых языков. Высота карточки
// растёт вместе с числом языков: иначе легенда наезжает на подпись.
func Languages(profile *stats.Profile) string {
	width, minHeight := cardWidth, cardHeight

	const (
		legendTop      = 64.0
		legendBottomIn = 26.0 // отступ под подпись о выборке
		preferredRow   = 26.0
		minimalRow     = 18.0
	)

	ranked := profile.RankedLanguages()
	total := profile.TotalBytes()

	rowHeight := preferredRow
	height := minHeight
	if count := float64(len(ranked)); count > 0 {
		rowHeight = math.Min(preferredRow, (minHeight-legendTop-legendBottomIn)/count)
		if rowHeight < minimalRow {
			rowHeight = minimalRow
			height = legendTop + rowHeight*count + legendBottomIn
		}
	}

	parts := []string{
		svgOpen(width, height),
		card(width, height),
		txtBold(24, 36, "Используемые языки", 15, colorText),
	}

	if len(ranked) == 0 || total == 0 {
		parts = append(parts,
			txtAnchor(width/2, height/2, "нет данных о языках", 13, colorMuted, "middle"),
			"</svg>")
		return strings.Join(parts, "\n")
	}

	cx := 116.0
	cy := legendTop + (height-legendTop-legendBottomIn)/2
	const rOuter, rInner = 78.0, 50.0
	angle := -math.Pi / 2

	for _, language := range ranked {
		sweep := language.Share(total) / 100 * tau
		color := languageColors[language.Name]
		if color == "" {
			color = colorMuted
		}
		parts = append(parts, fmt.Sprintf(
			`<path d="%s" fill="%s" stroke="%s" stroke-width="1"><title>%s: %.1f%%</title></path>`,
			donutPath(cx, cy, rOuter, rInner, angle, angle+sweep), color, colorCardBg,
			escape(language.Name), language.Share(total)))
		angle += sweep
	}

	parts = append(parts,
		txtAnchor(cx, cy-2, strconv.Itoa(len(ranked)), 24, colorText, "middle"),
		txtAnchor(cx, cy+18, "языков", 11, colorMuted, "middle"))

	const legendX = 228.0
	for index, language := range ranked {
		y := legendTop + rowHeight/2 + float64(index)*rowHeight
		color := languageColors[language.Name]
		if color == "" {
			color = colorMuted
		}
		parts = append(parts,
			fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="10" height="10" rx="2" fill="%s"/>`, legendX, y-9, color),
			txt(legendX+18, y, language.Name, 12, colorText),
			txtAnchor(width-24, y, fmt.Sprintf("%.1f%%", language.Share(total)), 12, colorMuted, "end"))
	}

	parts = append(parts,
		txt(24, height-14, "по всем репозиториям, включая приватные", 10, colorMuted),
		"</svg>")

	return strings.Join(parts, "\n")
}
