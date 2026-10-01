// statsgen собирает статистику GitHub-профиля и рисует её в SVG.
//
// Публичные демо-деплои сторонних виджетов недоступны
// (github-readme-stats.vercel.app не отвечает,
// github-readme-activity-graph.vercel.app отдаёт 402 DEPLOYMENT_DISABLED),
// поэтому графики строятся локально и коммитятся в репозиторий.
//
// Использование:
//
//	go run ./cmd/statsgen stats --user ViktorVanyuta --out assets
//	go run ./cmd/statsgen check-leaks
package main

import (
	"fmt"
	"os"
)

const usageText = `statsgen — генератор статистики GitHub-профиля

Использование:
  statsgen stats [--user LOGIN] [--out DIR]
      Считает коммиты за всё время и используемые языки по всем
      репозиториям, включая приватные, и рисует overview.svg и languages.svg.

      Токен берётся из GITHUB_TOKEN или GH_TOKEN. Без него приватные
      репозитории не видны и используется REST-фолбэк.

  statsgen check-leaks [--staged-only]
      Ищет в индексе, рабочем дереве и истории токены вида
      ghp_*, github_pat_*, gho_*, gh[su]_* и завершается с кодом 1 при найденном.

  statsgen banner [--out DIR] [--name NAME] [--role ROLE]
      Рисует шапку профиля (banner.svg). Токен не нужен.

  statsgen stack [--out DIR]
      Рисует карточку стека (stack.svg) с названиями технологий. Токен не нужен.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usageText)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "stats":
		err = runStats(os.Args[2:])
	case "check-leaks":
		err = runCheckLeaks(os.Args[2:])
	case "banner":
		err = runBanner(os.Args[2:])
	case "stack":
		err = runStack(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usageText)
		return
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда %q\n\n%s", os.Args[1], usageText)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}
