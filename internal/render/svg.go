// Package render рисует SVG-карточки профиля: шапку, стек, сводку и языки.
//
// Всё рисуется локально, без внешних сервисов: публичные виджеты для профиля
// недоступны, а локальные файлы не зависят от чужой вёрстки.
package render

import (
	"fmt"
	"strconv"
	"strings"
)

// Палитра GitHub Dark и общая геометрия. Карточки рисуются функцией card, а
// заголовки — cardTitle, поэтому фон, рамка и отступы у всех блоков совпадают.
const (
	colorCardBg = "#161b22"
	colorBorder = "#30363d"
	colorText   = "#e6edf3"
	colorMuted  = "#8b949e"
	colorGreen  = "#3fb950"
	colorBlue   = "#58a6ff"
	fontFamily  = "Verdana,DejaVu Sans,sans-serif"

	cardRadius = 10.0
)

// card — подложка карточки.
func card(width, height float64) string {
	return fmt.Sprintf(`<rect width="%.0f" height="%.0f" rx="%.0f" fill="%s" stroke="%s"/>`,
		width, height, cardRadius, colorCardBg, colorBorder)
}

func svgOpen(width, height float64) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`,
		width, height, width, height)
}

func escape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}

func svgText(x, y float64, value string, size int, fill, anchor, weight string) string {
	return fmt.Sprintf(
		`<text x="%.1f" y="%.1f" font-family="%s" font-size="%d" fill="%s" text-anchor="%s" font-weight="%s">%s</text>`,
		x, y, fontFamily, size, fill, anchor, weight, escape(value))
}

func txt(x, y float64, value string, size int, fill string) string {
	return svgText(x, y, value, size, fill, "start", "normal")
}

func txtBold(x, y float64, value string, size int, fill string) string {
	return svgText(x, y, value, size, fill, "start", "bold")
}

func txtAnchor(x, y float64, value string, size int, fill, anchor string) string {
	return svgText(x, y, value, size, fill, anchor, "normal")
}

func txtAnchorBold(x, y float64, value string, size int, fill, anchor string) string {
	return svgText(x, y, value, size, fill, anchor, "bold")
}

// formatNumber разделяет разряды неразрывным пробелом: 1 234.
func formatNumber(value int) string {
	digits := strconv.Itoa(value)
	var builder strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			builder.WriteByte(' ')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// estimateWidth — приблизительная ширина строки в Verdana: точной метрики без
// шрифтов нет, поэтому ширины типовых букв заложены с запасом.
func estimateWidth(value string, fontSize float64) float64 {
	width := 0.0
	for _, r := range value {
		switch {
		case r == ' ':
			width += 0.35
		case strings.ContainsRune("ilj.,:'|", r):
			width += 0.32
		case strings.ContainsRune("mwMW@", r):
			width += 0.92
		case r >= 'A' && r <= 'Z':
			width += 0.68
		default:
			width += 0.58
		}
	}
	return width * fontSize
}
