package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ViktorVanyuta/ViktorVanyuta/internal/githubapi"
	"github.com/ViktorVanyuta/ViktorVanyuta/internal/gitx"
	"github.com/ViktorVanyuta/ViktorVanyuta/internal/leaks"
	"github.com/ViktorVanyuta/ViktorVanyuta/internal/render"
	"github.com/ViktorVanyuta/ViktorVanyuta/internal/stats"
)

func runStats(args []string) error {
	flags := flag.NewFlagSet("stats", flag.ExitOnError)
	user := flags.String("user", envOr("GH_USER", "ViktorVanyuta"), "логин GitHub")
	out := flags.String("out", "assets", "каталог для SVG")
	if err := flags.Parse(args); err != nil {
		return err
	}

	token := firstNonEmpty(os.Getenv("GITHUB_TOKEN"), os.Getenv("GH_TOKEN"))
	client := githubapi.NewClient(token)

	var profile *stats.Profile
	var err error
	switch {
	case token != "":
		profile, err = client.FetchGraphQLProfile(*user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GraphQL недоступен (%s), переключаюсь на REST\n", err)
		}
	default:
		fmt.Fprintln(os.Stderr, "Токен не задан: приватные репозитории не видны, считаю по REST")
	}

	if profile == nil {
		profile, err = client.FetchRESTProfile(*user)
		if err != nil {
			return err
		}
	}

	if _, err := writeIfChanged(filepath.Join(*out, "overview.svg"), render.Overview(profile)); err != nil {
		return err
	}
	if _, err := writeIfChanged(filepath.Join(*out, "languages.svg"), render.Languages(profile)); err != nil {
		return err
	}

	fmt.Printf("Источник: %s\n", profile.Source)
	fmt.Printf("Коммитов всего: %d\n", profile.TotalCommits())
	fmt.Printf("Репозиториев: %d\n", profile.ReposTotal())
	fmt.Printf("Языков: %d\n", len(profile.Languages))
	return nil
}

// runBanner рисует шапку профиля. Токен не нужен: баннер не зависит от API.
func runBanner(args []string) error {
	flags := flag.NewFlagSet("banner", flag.ExitOnError)
	out := flags.String("out", "assets", "каталог для SVG")
	name := flags.String("name", "Viktor Vanyuta", "имя в шапке")
	role := flags.String("role", "Backend-разработчик Go", "роль под именем")
	if err := flags.Parse(args); err != nil {
		return err
	}

	info := render.BannerInfo{
		Name:  *name,
		Role:  *role,
		Techs: []string{"Go", "PostgreSQL", "gRPC", "Redis", "Docker", "Traefik"},
	}

	return writeAsset(filepath.Join(*out, "banner.svg"), render.Banner(info))
}

// runStack рисует карточку стека. Токен не нужен: список технологий задан
// вручную в render.DeclaredStack.
func runStack(args []string) error {
	flags := flag.NewFlagSet("stack", flag.ExitOnError)
	out := flags.String("out", "assets", "каталог для SVG")
	if err := flags.Parse(args); err != nil {
		return err
	}

	return writeAsset(filepath.Join(*out, "stack.svg"), render.Stack(render.DeclaredStack()))
}

func runCheckLeaks(args []string) error {
	flags := flag.NewFlagSet("check-leaks", flag.ExitOnError)
	stagedOnly := flags.Bool("staged-only", false, "проверять только проиндексированные изменения")
	if err := flags.Parse(args); err != nil {
		return err
	}

	var findings []leaks.Finding

	if *stagedOnly {
		diff, err := gitx.Output("diff", "--cached")
		if err != nil {
			return err
		}
		findings = leaks.Scan("index (git diff --cached)", diff)
	} else {
		tracked, err := gitx.Lines("ls-files")
		if err != nil {
			return err
		}
		findings = append(findings, leaks.ScanFiles(tracked)...)

		untracked, err := gitx.Lines("ls-files", "--others", "--exclude-standard")
		if err != nil {
			return err
		}
		findings = append(findings, leaks.ScanFiles(untracked)...)

		history, err := gitx.Output("log", "-p", "--all")
		if err != nil {
			return err
		}
		findings = append(findings, leaks.Scan("история коммитов (git log -p)", history)...)
	}

	if leaks.ReportFindings(findings) {
		return leaks.ErrLeaksFound
	}

	if *stagedOnly {
		fmt.Println("Токенов не найдено в проиндексированных изменениях.")
	} else {
		fmt.Println("Токенов не найдено в рабочем дереве и истории.")
	}
	return nil
}

// writeAsset сохраняет SVG и сообщает, изменился ли файл.
func writeAsset(path, content string) error {
	changed, err := writeIfChanged(path, content)
	if err != nil {
		return err
	}
	if changed {
		fmt.Printf("Обновлён %s\n", path)
	} else {
		fmt.Printf("%s не изменился\n", path)
	}
	return nil
}

// writeIfChanged пишет файл, только если содержимое отличается: так генерация
// не трогает mtime и не плодит пустые коммиты.
func writeIfChanged(path, content string) (bool, error) {
	if existing, err := os.ReadFile(path); err == nil && string(existing) == content {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
