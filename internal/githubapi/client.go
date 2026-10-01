// Package githubapi считает статистику профиля через GitHub API: коммиты —
// GraphQL contributionsCollection, репозитории и языки — REST, включая
// приватные.
//
// Токен принимается конструктором и живёт только в заголовке Authorization:
// наружу он не попадает, а в тексте ошибок заменяется на маску.
package githubapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ViktorVanyuta/ViktorVanyuta/internal/stats"
)

// Адреса API — переменные, чтобы тесты могли подменить их на httptest-сервер.
var (
	restAPI    = "https://api.github.com"
	graphqlAPI = "https://api.github.com/graphql"
)

// githubError — ошибка обращения к GitHub API. Значение токена в неё не попадает.
type githubError struct{ msg string }

func (e *githubError) Error() string { return e.msg }

func githubErrorf(format string, args ...any) error {
	return &githubError{msg: fmt.Sprintf(format, args...)}
}

// String скрывает токен: клиент можно печатать в отладке через %v, не выводя
// секрет в лог. Метод на значении, а не на указателе, чтобы редактирование не
// работало для самого Client.
func (c Client) String() string {
	if c.token == "" {
		return "githubapi.Client{token: пусто}"
	}
	return "githubapi.Client{token: ***REDACTED***}"
}

// redact гарантирует, что значение токена не попадёт ни в вывод, ни в лог.
func redact(message, token string) string {
	if len(token) >= 8 && strings.Contains(message, token) {
		return strings.ReplaceAll(message, token, "***REDACTED***")
	}
	return message
}

func truncate(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit]
}

// Client — минимальный клиент GitHub API на стандартной библиотеке.
type Client struct {
	token string
	http  *http.Client

	// privateVisible ставится, когда список репозиториев удалось получить
	// с токеном. Помечает, можно ли доверять приватной части статистики.
	privateVisible bool
}

func NewClient(token string) *Client {
	return &Client{token: token, http: &http.Client{Timeout: 45 * time.Second}}
}

func (c *Client) request(endpoint string, payload any, out any) error {
	var body io.Reader
	method := http.MethodGet
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("кодирование запроса: %w", err)
		}
		body = bytes.NewReader(encoded)
		method = http.MethodPost
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return fmt.Errorf("создание запроса: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "readme-stats-generator")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return githubErrorf("%s: %s", endpoint, redact(err.Error(), c.token))
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return githubErrorf("%s: чтение ответа: %s", endpoint, redact(err.Error(), c.token))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := redact(truncate(string(data), 300), c.token)
		return githubErrorf("%s: HTTP %d: %s", endpoint, resp.StatusCode, detail)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return githubErrorf("%s: разбор JSON: %s", endpoint, redact(err.Error(), c.token))
	}
	return nil
}

// paginate обходит постраничные ответы REST API до maxPages страниц.
func (c *Client) paginate(path string, maxPages int) ([]json.RawMessage, error) {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}

	var items []json.RawMessage
	for page := 1; page <= maxPages; page++ {
		var batch []json.RawMessage
		endpoint := fmt.Sprintf("%s%sper_page=100&page=%d", path, separator, page)
		if err := c.request(endpoint, nil, &batch); err != nil {
			return nil, err
		}
		items = append(items, batch...)
		if len(batch) < 100 {
			return items, nil
		}
	}
	return items, nil
}

// Repo — репозиторий в виде, в котором его отдаёт REST API.
type Repo struct {
	FullName string `json:"full_name"`
	Fork     bool   `json:"fork"`
	Private  bool   `json:"private"`
}

type commitItem struct {
	Commit struct {
		Author struct {
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// publicReposPath — список репозиториев, доступных всем. Приватные сюда не
// попадают даже при наличии токена.
func (c *Client) publicReposPath(login string) string {
	return restAPI + "/users/" + url.PathEscape(login) + "/repos?sort=full_name"
}

// ownRepos — репозитории владельца без форков: чужой код в статистику не идёт.
//
// С токеном список берётся через /user/repos, потому что публичный
// /users/{login}/repos приватные репозитории не возвращает. Если токен
// отвергнут или не имеет нужных прав, сваливается на публичный список, чтобы
// не терять статистику целиком.
func (c *Client) ownRepos(login string) ([]Repo, error) {
	path := c.publicReposPath(login)
	if c.token != "" {
		path = restAPI + "/user/repos?affiliation=owner&sort=full_name"
	}

	raw, err := c.paginate(path, 20)
	authenticated := err == nil && c.token != ""
	if err != nil && c.token != "" {
		fmt.Fprintf(os.Stderr,
			"  %s: приватные репозитории недоступны (%s), считаю только публичные\n", path, err)
		// Повтор идёт без заголовка Authorization: отвергнутый токен GitHub
		// отклоняет и публичные запросы, а не только приватные.
		anonymous := &Client{http: c.http}
		raw, err = anonymous.paginate(c.publicReposPath(login), 20)
	}
	if err != nil {
		return nil, err
	}
	c.privateVisible = authenticated

	own := make([]Repo, 0, len(raw))
	for _, item := range raw {
		var repo Repo
		if err := json.Unmarshal(item, &repo); err != nil {
			return nil, githubErrorf("разбор репозитория: %s", redact(err.Error(), c.token))
		}
		if !repo.Fork {
			own = append(own, repo)
		}
	}
	return own, nil
}

// fillFromRepos заполняет счётчики репозиториев и языков по списку репозиториев.
func (c *Client) fillFromRepos(profile *stats.Profile, own []Repo) error {
	languages := make(map[string]int)
	failed := 0
	for _, repo := range own {
		var sizes map[string]int
		if err := c.request(restAPI+"/repos/"+repo.FullName+"/languages", nil, &sizes); err != nil {
			fmt.Fprintf(os.Stderr, "  %s: языки пропущены (%s)\n", repo.FullName, err)
			failed++
			continue
		}
		for language, size := range sizes {
			languages[language] += size
		}
	}

	private := 0
	for _, repo := range own {
		if repo.Private {
			private++
		}
	}

	profile.Languages = languages
	profile.ReposPublic = len(own) - private
	profile.ReposPrivate = private

	fmt.Printf("Репозиториев в выборке: %d (приватных: %d)\n", len(own), private)
	fmt.Printf("Языков: %d\n", len(languages))
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "  пропущено репозиториев: %d\n", failed)
	}
	if !c.privateVisible {
		fmt.Fprintln(os.Stderr,
			"ВНИМАНИЕ: приватные репозитории не видны — нужен личный PAT со scope `repo`.")
	} else if private == 0 {
		fmt.Fprintln(os.Stderr,
			"ВНИМАНИЕ: токен не видит ни одного приватного репозитория — проверь scope `repo`.")
	}
	return nil
}

// profileQuery запрашивает только список годов: счётчики репозиториев берутся
// из REST-списка, который заодно нужен для языков и позволяет исключить форки
// (в GraphQL repositories форки входят в totalCount).
const profileQuery = `
query($login: String!) {
  user(login: $login) {
    contributionsCollection { contributionYears }
  }
}
`

const commitsByYearQuery = `
query($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    contributionsCollection(from: $from, to: $to) { totalCommitContributions }
  }
}
`

type gqlEnvelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *Client) graphql(query string, vars map[string]any, out any) error {
	var envelope gqlEnvelope
	if err := c.request(graphqlAPI, map[string]any{"query": query, "variables": vars}, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		messages := make([]string, 0, len(envelope.Errors))
		for _, item := range envelope.Errors {
			messages = append(messages, item.Message)
		}
		return githubErrorf("GraphQL: %s", redact(strings.Join(messages, "; "), c.token))
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return githubErrorf("GraphQL: разбор ответа: %s", redact(err.Error(), c.token))
	}
	return nil
}

type gqlProfileResponse struct {
	User *struct {
		PublicRepositories      struct{ TotalCount int } `json:"publicRepositories"`
		PrivateRepositories     struct{ TotalCount int } `json:"privateRepositories"`
		ContributionsCollection struct {
			ContributionYears []int `json:"contributionYears"`
		} `json:"contributionsCollection"`
	} `json:"user"`
}

type gqlYearResponse struct {
	User struct {
		ContributionsCollection struct {
			TotalCommitContributions int `json:"totalCommitContributions"`
		} `json:"contributionsCollection"`
	} `json:"user"`
}

// FetchGraphQLProfile считает коммиты за всё время через contributionsCollection:
// отдельный запрос на каждый год с года регистрации аккаунта. Такой подход не
// упирается в лимит пагинации и учитывает приватные репозитории.
func (c *Client) FetchGraphQLProfile(login string) (*stats.Profile, error) {
	var data gqlProfileResponse
	if err := c.graphql(profileQuery, map[string]any{"login": login}, &data); err != nil {
		return nil, err
	}
	if data.User == nil {
		return nil, githubErrorf("пользователь %s не найден", login)
	}

	years := data.User.ContributionsCollection.ContributionYears
	if len(years) == 0 {
		return nil, githubErrorf("contributionYears пуст")
	}
	sort.Ints(years)

	commits := make(map[string]int, len(years))
	for _, year := range years {
		vars := map[string]any{
			"login": login,
			"from":  fmt.Sprintf("%04d-01-01T00:00:00Z", year),
			"to":    fmt.Sprintf("%04d-12-31T23:59:59Z", year),
		}
		var yearData gqlYearResponse
		if err := c.graphql(commitsByYearQuery, vars, &yearData); err != nil {
			return nil, err
		}
		count := yearData.User.ContributionsCollection.TotalCommitContributions
		commits[fmt.Sprintf("%04d", year)] = count
		fmt.Printf("  %d: коммитов %d\n", year, count)
	}

	profile := &stats.Profile{
		Login:         login,
		CommitsByYear: commits,
		Source:        "GraphQL contributionsCollection",
	}
	own, err := c.ownRepos(login)
	if err != nil {
		return nil, err
	}
	return profile, c.fillFromRepos(profile, own)
}

// FetchRESTProfile — резервный путь без GraphQL: коммиты по месяцам из списка
// репозиториев. Работает медленнее и считает только последние коммиты каждого
// репозитория, поэтому используется лишь если GraphQL недоступен.
func (c *Client) FetchRESTProfile(login string) (*stats.Profile, error) {
	own, err := c.ownRepos(login)
	if err != nil {
		return nil, err
	}

	commitsByYear := make(map[string]int)
	for _, repo := range own {
		raw, err := c.paginate(restAPI+"/repos/"+repo.FullName+"/commits", 10)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: пропущен (%s)\n", repo.FullName, err)
			continue
		}
		for _, item := range raw {
			var commit commitItem
			if err := json.Unmarshal(item, &commit); err != nil {
				continue
			}
			if date := commit.Commit.Author.Date; len(date) >= 4 {
				commitsByYear[date[:4]]++
			}
		}
	}

	source := "REST /repos/../commits"
	if !c.privateVisible {
		source += " (без приватных репозиториев)"
	}

	profile := &stats.Profile{
		Login:         login,
		CommitsByYear: commitsByYear,
		Source:        source,
	}
	return profile, c.fillFromRepos(profile, own)
}
