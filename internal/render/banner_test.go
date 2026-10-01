package render

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	bannerRectPattern = regexp.MustCompile(`<rect x="([\d.]+)" y="([\d.]+)" width="([\d.]+)" height="([\d.]+)"`)
	// bannerChipPattern отсеивает карточку и верхнюю полосу по радиусу скругления.
	bannerChipPattern = regexp.MustCompile(`<rect x="([\d.]+)" y="([\d.]+)" width="([\d.]+)" height="([\d.]+)" rx="13"`)
	bannerTextPattern = regexp.MustCompile(`<text x="([\d.]+)" y="([\d.]+)"[^>]*font-size="(\d+)"[^>]*text-anchor="([a-z]+)"[^>]*>([^<]*)</text>`)
)

func testBanner() BannerInfo {
	return BannerInfo{
		Name:  "Viktor Vanyuta",
		Role:  "Backend-разработчик Go",
		Techs: []string{"Go", "PostgreSQL", "gRPC", "Redis", "Docker", "Traefik"},
	}
}

func TestBannerStructure(t *testing.T) {
	svg := Banner(testBanner())

	for _, want := range []string{
		`<svg`, `</svg>`, `viewBox="0 0 1000 280"`,
		`linearGradient id="bg"`, `linearGradient id="name"`, `linearGradient id="accent"`,
		`role="img"`, `aria-label=`,
		"Viktor Vanyuta",
		"Backend-разработчик Go",
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("в баннере нет %q", want)
		}
	}

	// Каждая используемая ссылка на градиент должна быть объявлена.
	for _, id := range regexp.MustCompile(`url\(#([\w-]+)\)`).FindAllStringSubmatch(svg, -1) {
		if !strings.Contains(svg, `id="`+id[1]+`"`) {
			t.Errorf("ссылка на несуществующий градиент #%s", id[1])
		}
	}
}

func TestBannerChipsStayInsideCard(t *testing.T) {
	chips := bannerChipRects(Banner(testBanner()))

	if len(chips) != len(testBanner().Techs) {
		t.Errorf("плашек %d, ожидалось %d", len(chips), len(testBanner().Techs))
	}
	for _, chip := range chips {
		if chip.x < 16 {
			t.Errorf("плашка x=%.1f ближе 16 к левому краю", chip.x)
		}
		if chip.x+chip.width > 984 {
			t.Errorf("плашка x=%.1f width=%.1f выходит за карточку", chip.x, chip.width)
		}
	}
}

func TestBannerTextFitsWidth(t *testing.T) {
	svg := Banner(testBanner())

	for _, item := range bannerTextPattern.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.ParseFloat(item[1], 64)
		size, _ := strconv.Atoi(item[3])
		anchor := item[4]
		value := item[5]

		textWidth := estimateWidth(value, float64(size))
		var left, right float64
		switch anchor {
		case "middle":
			left, right = x-textWidth/2, x+textWidth/2
		case "end":
			left, right = x-textWidth, x
		default:
			left, right = x, x+textWidth
		}

		if left < 8 || right > 992 {
			t.Errorf("текст %q (%.0fpx, x=%.0f, anchor=%s) выходит за карточку: [%.1f, %.1f]",
				value, textWidth, x, anchor, left, right)
		}
	}
}

func TestBannerNameIsCenteredAndBold(t *testing.T) {
	svg := Banner(testBanner())
	if !strings.Contains(svg, `font-weight="bold"`) {
		t.Error("имя должно быть жирным")
	}

	match := regexp.MustCompile(`<text x="([\d.]+)" y="([\d.]+)"[^>]*font-size="58"[^>]*text-anchor="middle"[^>]*>([^<]*)</text>`).
		FindStringSubmatch(svg)
	if match == nil {
		t.Fatalf("имя не найдено в баннере:\n%s", svg)
	}
	if x, _ := strconv.ParseFloat(match[1], 64); x != 500 {
		t.Errorf("имя выровнено по x=%.0f, ожидался центр 500", x)
	}
}

func TestBannerWrapsLongTechRow(t *testing.T) {
	long := []string{
		"Go", "PostgreSQL", "MySQL", "ClickHouse", "Kafka", "RabbitMQ", "NATS",
		"Docker", "Terraform", "Ansible", "Helm", "Prometheus",
		"Loki", "Traefik", "Nginx", "Envoy", "Redis", "etcd",
	}

	svg := Banner(BannerInfo{Name: "A B", Role: "C D", Techs: long})

	// Плашки отличаются от карточки и верхней полосы радиусом скругления 13.
	chipRects := bannerChipRects(svg)
	if len(chipRects) != len(long) {
		t.Errorf("плашек %d, ожидалось %d", len(chipRects), len(long))
	}

	rows := map[float64]bool{}
	for _, chipRect := range chipRects {
		rows[chipRect.y] = true
		if chipRect.x < 16 || chipRect.x+chipRect.width > 984 {
			t.Errorf("плашка x=%.1f width=%.1f выходит за карточку", chipRect.x, chipRect.width)
		}
	}
	if len(rows) < 2 {
		t.Errorf("длинный ряд должен переноситься, строк: %d", len(rows))
	}
}

// TestSplitIntoRows — раскладка не должна терять и не рвать плашки.
func TestSplitIntoRows(t *testing.T) {
	techs := []string{"aa", "bb", "cc"}
	widths := []float64{60, 60, 60}

	rows := splitIntoRows(techs, widths, 10, 140)
	if len(rows) != 2 {
		t.Fatalf("строк: %d, ожидалось 2: %+v", len(rows), rows)
	}
	if len(rows[0].items) != 2 || len(rows[1].items) != 1 {
		t.Errorf("раскладка неверна: %+v", rows)
	}

	total := 0
	for _, row := range rows {
		total += len(row.items)
	}
	if total != len(techs) {
		t.Errorf("в строках %d плашек, ожидалось %d", total, len(techs))
	}
}

func TestSplitIntoRowsFitsEverything(t *testing.T) {
	techs := []string{"a", "b", "c", "d"}
	widths := []float64{40, 40, 40, 40}

	rows := splitIntoRows(techs, widths, 10, 1000)
	if len(rows) != 1 || len(rows[0].items) != 4 {
		t.Errorf("в одну строку всё не поместилось: %+v", rows)
	}
}

func TestEstimateWidthGrowsWithText(t *testing.T) {
	short := estimateWidth("Go", 13)
	long := estimateWidth("PostgreSQL", 13)
	if long <= short {
		t.Errorf("«PostgreSQL» (%.1f) должен быть шире «Go» (%.1f)", long, short)
	}
	if bigger := estimateWidth("Go", 26); bigger <= short {
		t.Errorf("крупный кегль должен давать большую ширину: %.1f против %.1f", bigger, short)
	}
}

func TestBannerWithoutTechs(t *testing.T) {
	svg := Banner(BannerInfo{Name: "A", Role: "B"})
	if strings.Contains(svg, `rx="13"`) {
		t.Errorf("без технологий плашек быть не должно:\n%s", svg)
	}
}

type chipRect struct{ x, y, width float64 }

// bannerChipRects выбирает из баннера прямоугольники-плашки, отсекая карточку
// и верхнюю акцентную полосу по радиусу скругления.
func bannerChipRects(svg string) []chipRect {
	var chips []chipRect
	for _, match := range bannerChipPattern.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.ParseFloat(match[1], 64)
		y, _ := strconv.ParseFloat(match[2], 64)
		width, _ := strconv.ParseFloat(match[3], 64)
		chips = append(chips, chipRect{x: x, y: y, width: width})
	}
	return chips
}
