package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	colorInk        = "#0D1117"
	minMarkContrast = 4.5
)

// StackItem — одна технология в карточке стека.
type StackItem struct {
	Name  string
	Color string // фирменный цвет бренда
	Mark  string // короткий знак внутри плашки
}

// StackGroup — категория технологий с общим заголовком.
type StackGroup struct {
	Title string
	Items []StackItem
}

// DeclaredStack — стек, заявленный в профиле. Порядок групп значим: сначала
// язык и обмен данными, потом данные, потом инфраструктура.
func DeclaredStack() []StackGroup {
	return []StackGroup{
		{"Язык и фреймворки", []StackItem{
			{"Go (Golang)", "#00ADD8", "GO"},
			{"Gin", "#008ECF", "GI"},
		}},
		{"API и контракты", []StackItem{
			{"gRPC", "#2DA6B0", "GR"},
			{"Protobuf", "#4B6BFB", "PB"},
			{"GraphQL", "#E10098", "GQ"},
			{"gqlgen", "#B5379B", "GG"},
		}},
		{"Данные", []StackItem{
			{"PostgreSQL", "#336791", "PG"},
			{"Redis", "#D82C20", "RD"},
			{"Goose", "#D9A404", "GS"},
		}},
		{"Архитектура и безопасность", []StackItem{
			{"Clean Architecture", "#6E7681", "CA"},
			{"JWT Auth", "#374151", "JW"},
		}},
		{"Инфраструктура", []StackItem{
			{"Docker Compose", "#2496ED", "DC"},
			{"Traefik", "#24A1C1", "TF"},
			{"On-Premise File System", "#4C7A8C", "FS"},
		}},
	}
}

// Stack рисует стек как сгруппированные плашки: цвет бренда, короткий
// знак и полное название. Логотипов нет намеренно — публичные CDN знают
// меньше половины этих технологий, и ряд с дырами выглядел бы хуже.
func Stack(groups []StackGroup) string {
	const (
		width    = 760.0
		padding  = 28.0
		chipH    = 32.0
		chipGap  = 10.0
		rowGap   = 10.0
		groupGap = 22.0
		markSize = 22.0
		markPadX = 9.0
		textPad  = 12.0
		fontSize = 13.0
		markFont = 11.0
	)

	parts := []string{
		svgOpen(width, 400),
		card(width, 400),
		txtBold(padding, 40, "Стек технологий", 16, colorText),
	}

	y := 66.0
	for groupIndex, group := range groups {
		if groupIndex > 0 {
			y += groupGap
		}
		parts = append(parts, txt(padding, y, group.Title, 11, colorMuted))
		y += 14

		available := width - padding*2
		lineWidth := 0.0
		for _, item := range group.Items {
			chipWidth := markPadX + markSize + 8 + estimateWidth(item.Name, fontSize) + textPad

			if lineWidth > 0 && lineWidth+chipGap+chipWidth > available {
				y += chipH + rowGap
				lineWidth = 0
			}
			if lineWidth > 0 {
				lineWidth += chipGap
			}

			// Фирменный цвет не искажается: по нему красится маркер и рамка плашки,
			// а фон берётся лёгким тинтом, чтобы название читалось.
			x := padding + lineWidth
			parts = append(parts,
				fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s" fill-opacity="0.14" stroke="%s" stroke-opacity="0.5"/>`,
					x, y, chipWidth, chipH, chipH/2, item.Color, item.Color),
				fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="6" fill="%s"/>`,
					x+markPadX, y+(chipH-markSize)/2, markSize, markSize, item.Color),
				txtAnchor(x+markPadX+markSize/2, y+chipH/2+markFont/3, item.Mark, int(markFont), readableOn(item.Color), "middle"),
				txt(x+markPadX+markSize+8, y+chipH/2+fontSize/3, item.Name, int(fontSize), colorText))

			lineWidth += chipWidth
		}
		y += chipH + rowGap
	}

	height := y + 8
	parts[0] = svgOpen(width, height)
	parts[1] = card(width, height)
	parts = append(parts, "</svg>")
	return strings.Join(parts, "\n")
}

// readableOn — цвет текста знака внутри маркера: белый или почти чёрный,
// в зависимости от контраста с цветом бренда. Цвет бренда при этом не
// меняется, в отличие от заливки всей плашки.
func readableOn(color string) string {
	r, g, b := parseHexColor(color)
	if contrastWithWhite(r, g, b) >= minMarkContrast {
		return "#FFFFFF"
	}
	if contrastWithBlack(r, g, b) >= minMarkContrast {
		return colorInk
	}
	// Ни один из полюсов не даёт нужного контраста — берём тот, что ближе.
	if contrastWithWhite(r, g, b) >= contrastWithBlack(r, g, b) {
		return "#FFFFFF"
	}
	return colorInk
}

// contrastWithBlack — контраст цвета бренда относительно почти чёрного текста.
func contrastWithBlack(r, g, b float64) float64 {
	return (relativeLuminance(r, g, b) + 0.05) / 0.05
}

// parseHexColor разбирает #RGB и #RRGGBB. Нераспознанное значение даёт серый:
// на карточке лучше нейтральная плашка, чем чёрная.
func parseHexColor(color string) (float64, float64, float64) {
	trimmed := strings.TrimPrefix(color, "#")
	if len(trimmed) == 3 {
		var expanded strings.Builder
		for _, r := range trimmed {
			expanded.WriteRune(r)
			expanded.WriteRune(r)
		}
		trimmed = expanded.String()
	}

	value, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil || len(trimmed) != 6 {
		return 128, 128, 128
	}
	return float64((value >> 16) & 0xFF), float64((value >> 8) & 0xFF), float64(value & 0xFF)
}

// contrastWithWhite — контраст цвета бренда относительно белого текста.
// Формула WCAG: (L1 + 0.05) / (L2 + 0.05), где L1 — светлее.
func contrastWithWhite(r, g, b float64) float64 {
	return 1.05 / (relativeLuminance(r, g, b) + 0.05)
}

func relativeLuminance(r, g, b float64) float64 {
	channel := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}
