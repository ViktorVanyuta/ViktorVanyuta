package render

import (
	"fmt"
	"strings"
)

// BannerInfo — содержимое шапки профиля.
type BannerInfo struct {
	Name  string
	Role  string
	Techs []string
}

// Banner рисует шапку профиля: имя с градиентом, роль и ряд технологий.
// Генерируется локально, чтобы README не зависел от сторонних картинок.
func Banner(info BannerInfo) string {
	const (
		width   = 1000.0
		height  = 280.0
		chipTop = 216.0
	)

	parts := []string{
		`<svg xmlns="http://www.w3.org/2000/svg" width="1000" height="280" viewBox="0 0 1000 280" role="img" aria-label="` + escape(info.Name+" — "+info.Role) + `">`,
		`<defs>
  <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
    <stop offset="0%" stop-color="#0d1117"/>
    <stop offset="55%" stop-color="#161b22"/>
    <stop offset="100%" stop-color="#0b1020"/>
  </linearGradient>
  <linearGradient id="name" x1="0" y1="0" x2="1" y2="0">
    <stop offset="0%" stop-color="#58a6ff"/>
    <stop offset="50%" stop-color="#a371f7"/>
    <stop offset="100%" stop-color="#3fb950"/>
  </linearGradient>
  <linearGradient id="accent" x1="0" y1="0" x2="1" y2="0">
    <stop offset="0%" stop-color="#58a6ff" stop-opacity="0"/>
    <stop offset="50%" stop-color="#a371f7"/>
    <stop offset="100%" stop-color="#3fb950" stop-opacity="0"/>
  </linearGradient>
</defs>`,
		fmt.Sprintf(`<rect width="1000" height="280" rx="16" fill="url(#bg)" stroke="%s"/>`, colorBorder),
		`<rect x="0" y="0" width="1000" height="4" rx="2" fill="url(#accent)"/>`,
		txtAnchorBold(width/2, 130, info.Name, 58, "url(#name)", "middle"),
		txtAnchor(width/2, 174, info.Role, 21, colorText, "middle"),
	}

	parts = append(parts, renderChips(info.Techs, chipTop, width)...)
	parts = append(parts, "</svg>")
	return strings.Join(parts, "\n")
}

// renderChips рисует технологии скруглёнными плашками в один ряд по центру.
func renderChips(techs []string, top, canvasWidth float64) []string {
	if len(techs) == 0 {
		return nil
	}

	const (
		fontSize  = 13.0
		chipPadX  = 14.0
		chipH     = 26.0
		gap       = 9.0
		minGapX   = 20.0
		chipColor = "#1c2333"
		textColor = "#a5b4c8"
	)

	widths := make([]float64, len(techs))
	total := gap * float64(len(techs)-1)
	for index, tech := range techs {
		widths[index] = estimateWidth(tech, fontSize) + chipPadX*2
		total += widths[index]
	}

	// Слишком длинный ряд переносится в несколько строк, чтобы не вылезти за карточку.
	rows := splitIntoRows(techs, widths, gap, canvasWidth-minGapX*2)

	var parts []string
	for rowIndex, row := range rows {
		rowWidth := gap * float64(len(row.items)-1)
		for _, item := range row.items {
			rowWidth += item.width
		}
		x := (canvasWidth - rowWidth) / 2
		y := top + float64(rowIndex)*(chipH+gap)

		for _, item := range row.items {
			parts = append(parts,
				fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="13" fill="%s" stroke="%s"/>`,
					x, y, item.width, chipH, chipColor, colorBorder),
				txt(x+chipPadX, y+18, item.text, int(fontSize), textColor))
			x += item.width + gap
		}
	}

	return parts
}

type chip struct {
	text  string
	width float64
}

type chipRow struct{ items []chip }

// splitIntoRows раскладывает плашки по строкам, не разрывая их.
func splitIntoRows(techs []string, widths []float64, gap, limit float64) []chipRow {
	var rows []chipRow
	current := chipRow{}
	used := 0.0

	for index, tech := range techs {
		extra := widths[index]
		if len(current.items) > 0 {
			extra += gap
		}
		if len(current.items) > 0 && used+extra > limit {
			rows = append(rows, current)
			current, used = chipRow{}, 0
			extra = widths[index]
		}
		current.items = append(current.items, chip{text: tech, width: widths[index]})
		used += extra
	}
	if len(current.items) > 0 {
		rows = append(rows, current)
	}
	return rows
}
