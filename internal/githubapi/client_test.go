package githubapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ViktorVanyuta/ViktorVanyuta/internal/stats"
)

// withTestAPI поднимает тестовый сервер вместо GitHub API.
func withTestAPI(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	oldREST, oldGraphQL := restAPI, graphqlAPI
	restAPI, graphqlAPI = server.URL, server.URL+"/graphql"
	t.Cleanup(func() { restAPI, graphqlAPI = oldREST, oldGraphQL })

	return &Client{token: "test-token", http: server.Client()}
}

func TestOwnReposUsesUserReposWithToken(t *testing.T) {
	var paths []string
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Write([]byte(`[{"full_name":"a/b","fork":false,"private":true}]`))
	})

	repos, err := client.ownRepos("b")
	if err != nil {
		t.Fatalf("ownRepos: %v", err)
	}
	if len(repos) != 1 || repos[0].FullName != "a/b" {
		t.Fatalf("получено %v, ожидался репозиторий a/b", repos)
	}
	if len(paths) != 1 || paths[0] != "/user/repos" {
		t.Errorf("запрошены %v, ожидался только /user/repos", paths)
	}
}

func TestOwnReposSkipsForks(t *testing.T) {
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"full_name":"a/own","fork":false},
			{"full_name":"a/fork","fork":true}
		]`))
	})

	repos, err := client.ownRepos("b")
	if err != nil {
		t.Fatalf("ownRepos: %v", err)
	}
	if len(repos) != 1 || repos[0].FullName != "a/own" {
		t.Errorf("получено %v, форк должен быть исключён", repos)
	}
}

// TestOwnReposFallsBackToPublicOnBadToken — раньше неверный токен ронял всю
// генерацию вместо того, чтобы посчитать хотя бы публичные репозитории.
func TestOwnReposFallsBackToPublicOnBadToken(t *testing.T) {
	type call struct{ path, authorization string }
	var calls []call
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, call{r.URL.Path, r.Header.Get("Authorization")})
		if r.URL.Path == "/user/repos" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Bad credentials"}`))
			return
		}
		w.Write([]byte(`[{"full_name":"a/public","fork":false,"private":false}]`))
	})

	repos, err := client.ownRepos("b")
	if err != nil {
		t.Fatalf("ownRepos: %v", err)
	}
	if len(repos) != 1 || repos[0].FullName != "a/public" {
		t.Fatalf("получено %v, ожидался публичный репозиторий", repos)
	}
	if client.privateVisible {
		t.Error("после отказа токена приватные репозитории не видны")
	}
	if len(calls) != 2 || calls[0].path != "/user/repos" || calls[1].path != "/users/b/repos" {
		t.Fatalf("запрошены %v, ожидался сброс на публичный список", calls)
	}
	if calls[1].authorization != "" {
		t.Errorf("повтор ушёл с Authorization=%q — отвергнутый токен снова даст 401",
			calls[1].authorization)
	}
}

func TestOwnReposWithoutTokenNeverAsksUserRepos(t *testing.T) {
	var paths []string
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Write([]byte(`[]`))
	})
	client.token = ""

	if _, err := client.ownRepos("b"); err != nil {
		t.Fatalf("ownRepos: %v", err)
	}
	for _, path := range paths {
		if strings.Contains(path, "/user/repos") {
			t.Errorf("без токена запрошен %s — приватные данные так не достать", path)
		}
	}
}

func TestRequestSendsAuthorizationWithoutToken(t *testing.T) {
	var got string
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		w.Write([]byte(`{}`))
	})

	if err := client.request(restAPI+"/x", nil, nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if got != "Bearer test-token" {
		t.Errorf("Authorization = %q, ожидался Bearer test-token", got)
	}

	client.token = ""
	if err := client.request(restAPI+"/x", nil, nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if got != "" {
		t.Errorf("без токена Authorization = %q, ожидалось пусто", got)
	}
}

func TestRequestErrorDoesNotLeakToken(t *testing.T) {
	token := "ghp_" + strings.Repeat("s", 36)
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Bad credentials for ` + token + `"}`))
	})
	client.token = token

	err := client.request(restAPI+"/x", nil, nil)
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("токен утёк в ошибку: %v", err)
	}
}

func TestGraphQLSurfacesErrors(t *testing.T) {
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"errors":[{"message":"Field 'nope' doesn't exist"}]}`))
	})

	err := client.graphql(profileQuery, map[string]any{"login": "b"}, nil)
	if err == nil || !strings.Contains(err.Error(), "doesn't exist") {
		t.Fatalf("ошибка GraphQL не проброшена: %v", err)
	}
}

func TestFetchGraphQLProfileCountsYears(t *testing.T) {
	var queries int
	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/languages") {
			w.Write([]byte(`{"Go":100}`))
			return
		}
		if r.URL.Path == "/user/repos" {
			w.Write([]byte(`[{"full_name":"a/one","fork":false,"private":false},
			                 {"full_name":"a/two","fork":false,"private":true}]`))
			return
		}

		var body struct {
			Query string `json:"query"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		switch {
		case strings.Contains(body.Query, "contributionYears"):
			w.Write([]byte(`{"data":{"user":{
				"contributionsCollection":{"contributionYears":[2022,2023]}
			}}}`))
		case strings.Contains(body.Query, "totalCommitContributions"):
			queries++
			w.Write([]byte(`{"data":{"user":{"contributionsCollection":{"totalCommitContributions":10}}}}`))
		default:
			w.Write([]byte(`{"errors":[{"message":"unexpected query"}]}`))
		}
	})

	profile, err := client.FetchGraphQLProfile("b")
	if err != nil {
		t.Fatalf("FetchGraphQLProfile: %v", err)
	}
	if queries != 2 {
		t.Errorf("запросов на годы: %d, ожидалось 2", queries)
	}
	if profile.TotalCommits() != 20 {
		t.Errorf("коммитов: %d, ожидалось 20", profile.TotalCommits())
	}
	// Счётчики репозиториев приходят из REST-списка, а не из GraphQL.
	if profile.ReposTotal() != 2 || profile.ReposPublic != 1 || profile.ReposPrivate != 1 {
		t.Errorf("репозиториев: %d (публичных %d, приватных %d), ожидалось 2 (1, 1)",
			profile.ReposTotal(), profile.ReposPublic, profile.ReposPrivate)
	}
}

func TestFillFromReposWarnsAboutMissingPrivateRepos(t *testing.T) {
	var output strings.Builder
	restore := captureStderr(&output)
	defer restore()

	client := withTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	profile := &stats.Profile{Login: "b"}
	if err := client.fillFromRepos(profile, []Repo{{FullName: "a/b"}}); err != nil {
		t.Fatalf("fillFromRepos: %v", err)
	}
	restore()

	if !strings.Contains(output.String(), "scope `repo`") {
		t.Errorf("нет предупреждения о приватных репозиториях:\n%s", output.String())
	}
	if profile.Languages["Go"] != 0 {
		t.Errorf("не должен ожидаться Go: %v", profile.Languages)
	}
}

func TestClientStringHidesToken(t *testing.T) {
	const secret = "test-token-value"
	client := NewClient(secret)

	// Клиент часто печатают в отладке: %v, %+v и %s обязаны вести себя одинаково.
	for _, format := range []string{"%v", "%+v", "%s"} {
		if out := fmt.Sprintf(format, client); strings.Contains(out, secret) {
			t.Errorf("%s вывел токен: %q", format, out)
		}
	}
	if out := fmt.Sprintf("%+v", *client); strings.Contains(out, secret) {
		t.Errorf("значение клиента раскрывает токен: %q", out)
	}
}
