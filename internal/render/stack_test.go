package render

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	stackChipPattern = regexp.MustCompile(`<rect x="([\d.]+)" y="([\d.]+)" width="([\d.]+)" height="32\.0" rx="16\.0" fill="(#[0-9A-Fa-f]{6})" fill-opacity="0\.14"`)
	stackMarkPattern = regexp.MustCompile(`<text x="([\d.]+)" y="([\d.]+)"[^>]*font-size="11"[^>]*text-anchor="middle"[^>]*>([A-Za-z]{1,3})</text>`)
)

func TestStackContainsWholeDeclaredStack(t *testing.T) {
	svg := Stack(DeclaredStack())

	want := []string{
		"Go (Golang)", "Gin", "gRPC", "Protobuf", "GraphQL", "gqlgen",
		"PostgreSQL", "Redis", "Goose", "Clean Architecture", "JWT Auth",
		"Docker Compose", "Traefik", "On-Premise File System",
	}
	for _, name := range want {
		if !strings.Contains(svg, ">"+name+"<") {
			t.Errorf("в карточке стека нет %q", name)
		}
	}

	for _, group := range DeclaredStack() {
		if !strings.Contains(svg, ">"+group.Title+"<") {
			t.Errorf("нет заголовка группы %q", group.Title)
		}
	}
}

func TestStackHasOneChipPerTechnology(t *testing.T) {
	svg := Stack(DeclaredStack())

	total := 0
	for _, group := range DeclaredStack() {
		total += len(group.Items)
	}
	if chips := len(stackChipPattern.FindAllStringSubmatch(svg, -1)); chips != total {
		t.Errorf("плашек %d, технологий %d", chips, total)
	}
	if marks := len(stackMarkPattern.FindAllStringSubmatch(svg, -1)); marks != total {
		t.Errorf("знаков %d, технологий %d", marks, total)
	}
}

func TestStackStaysInsideCard(t *testing.T) {
	svg := Stack(DeclaredStack())

	height := svgHeight(t, svg)
	const width, padding = 760.0, 28.0

	for _, chip := range stackChipPattern.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.ParseFloat(chip[1], 64)
		y, _ := strconv.ParseFloat(chip[2], 64)
		w, _ := strconv.ParseFloat(chip[3], 64)

		if x < padding-0.5 {
			t.Errorf("плашка x=%.1f за левым полем", x)
		}
		if x+w > width-padding+0.5 {
			t.Errorf("плашка x=%.1f width=%.1f выходит за карточку", x, w)
		}
		if y+32 > height-4 {
			t.Errorf("плашка y=%.1f упирается в низ карточки высотой %.0f", y, height)
		}
	}
}

func renderStackForTest(t *testing.T) string {
	t.Helper()
	return Stack(DeclaredStack())
}

func TestStackRowsDoNotOverlap(t *testing.T) {
	type box struct {
		x0, x1, y float64
	}
	var boxes []box

	for _, chip := range stackChipPattern.FindAllStringSubmatch(renderStackForTest(t), -1) {
		x, _ := strconv.ParseFloat(chip[1], 64)
		y, _ := strconv.ParseFloat(chip[2], 64)
		w, _ := strconv.ParseFloat(chip[3], 64)
		boxes = append(boxes, box{x0: x, x1: x + w, y: y})
	}

	for i := 0; i < len(boxes); i++ {
		for j := i + 1; j < len(boxes); j++ {
			a, b := boxes[i], boxes[j]
			if math.Abs(a.y-b.y) < 1 {
				if a.x0 < b.x1 && b.x0 < a.x1 {
					t.Errorf("плашки накладываются: [%.1f..%.1f] и [%.1f..%.1f] на y=%.1f",
						a.x0, a.x1, b.x0, b.x1, a.y)
				}
			}
		}
	}
}

func svg_stack(t *testing.T) string {
	t.Helper()
	return Stack(DeclaredStack())
}

// TestStackKeepsBrandColors — цвета брендов не должны искажаться ради контраста:
// иначе Go перестаёт быть «бирюзовым Go».
func TestStackKeepsBrandColors(t *testing.T) {
	svg := Stack(DeclaredStack())

	for _, group := range DeclaredStack() {
		for _, item := range group.Items {
			color := strings.ToUpper(item.Color)
			if !strings.Contains(svg, `fill="`+color+`" fill-opacity="0.14"`) {
				t.Errorf("цвет бренда %s для %q искажён или не найден", color, item.Name)
			}
		}
	}
}

// TestStackMarkContrast — знак внутри маркера обязан читаться: контраст с
// цветом бренда не ниже 4.5:1.
func TestStackMarkContrast(t *testing.T) {
	for _, group := range DeclaredStack() {
		for _, item := range group.Items {
			r, g, b := parseHexColor(item.Color)
			ink := readableOn(item.Color)

			var contrast float64
			switch ink {
			case "#FFFFFF":
				contrast = contrastWithWhite(r, g, b)
			case colorInk:
				contrast = contrastWithBlack(r, g, b)
			default:
				t.Fatalf("неожиданный цвет знака %s", ink)
			}

			if contrast < minMarkContrast {
				t.Errorf("%q: контраст знака %.2f при цвете %s", item.Name, contrast, item.Color)
			}
		}
	}
}

func TestStackNamesUseReadableInk(t *testing.T) {
	svg := Stack(DeclaredStack())
	if !strings.Contains(svg, ">Go (Golang)</text>") {
		t.Fatalf("название не отрисовано:\n%s", svg)
	}
	// Название плашки рисуется светлым цветом, а не белым по бренду.
	for _, match := range regexp.MustCompile(
		`<text x="[\d.]+" y="[\d.]+"[^>]*font-size="13" fill="([#0-9A-Fa-f]{6})"[^>]*>([^<]+)</text>`).
		FindAllStringSubmatch(svg, -1) {
		if strings.HasPrefix(match[2], "Goose") && match[1] != colorText {
			t.Errorf("название %q нарисовано цветом %s вместо %s", match[2], match[1], colorText)
		}
	}
}

func TestStackWrapsOnNarrowCanvas(t *testing.T) {
	long := []StackGroup{{
		Title: "Очень длинная группа",
		Items: []StackItem{
			{"Технология с очень длинным названием", "#00ADD8", "TX"},
			{"Ещё одна длинная технология", "#E10098", "TY"},
			{"Довольно длинное имя технологии", "#4B6BFB", "TZ"},
			{"И ещё одна длинная технология", "#336791", "TW"},
		},
	}}

	svg := Stack(long)
	chips := stackChipPattern.FindAllStringSubmatch(svg, -1)
	if len(chips) != len(long[0].Items) {
		t.Fatalf("плашек %d, ожидалось %d", len(chips), len(long[0].Items))
	}

	rows := map[float64]bool{}
	for _, chip := range chips {
		y, _ := strconv.ParseFloat(chip[2], 64)
		rows[y] = true
	}
	if len(rows) < 2 {
		t.Errorf("длинные названия должны переноситься, строк: %d", len(rows))
	}
}

func TestStackWithoutGroups(t *testing.T) {
	svg := Stack(nil)
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
		t.Errorf("пустой стек должен давать валидный каркас:\n%s", svg)
	}
	if len(stackChipPattern.FindAllStringSubmatch(svg, -1)) != 0 {
		t.Errorf("в пустом стеке не должно быть плашек:\n%s", svg)
	}
}

func TestParseHexColor(t *testing.T) {
	cases := map[string][3]float64{
		"#00ADD8": {0, 173, 216},
		"00ADD8":  {0, 173, 216},
		"#fff":    {255, 255, 255},
		"#000000": {0, 0, 0},
		"мусор":   {128, 128, 128},
	}
	for input, want := range cases {
		r, g, b := parseHexColor(input)
		if r != want[0] || g != want[1] || b != want[2] {
			t.Errorf("parseHexColor(%q) = %.0f,%.0f,%.0f, ожидалось %v", input, r, g, b, want)
		}
	}
}

func TestContrastHelpers(t *testing.T) {
	if got := contrastWithWhite(255, 255, 255); math.Abs(got-1) > 0.001 {
		t.Errorf("контраст белого с белым = %.3f, ожидалось 1", got)
	}
	if got := contrastWithWhite(0, 0, 0); math.Abs(got-21) > 0.1 {
		t.Errorf("контраст чёрного с белым = %.3f, ожидалось 21", got)
	}
	if got := contrastWithBlack(255, 255, 255); math.Abs(got-21) > 0.1 {
		t.Errorf("контраст белого с чёрным = %.3f, ожидалось 21", got)
	}
}

func TestReadableOnPicksReadableInk(t *testing.T) {
	if ink := readableOn("#FFFFFF"); ink != colorInk {
		t.Errorf("на белом нужен тёмный знак, получено %s", ink)
	}
	if ink := readableOn("#000000"); ink != "#FFFFFF" {
		t.Errorf("на чёрном нужен светлый знак, получено %s", ink)
	}
}
