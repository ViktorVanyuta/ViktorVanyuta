package githubapi

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	token := "ghp_" + strings.Repeat("x", 36)
	message := "запрос упал: Authorization Bearer " + token
	got := redact(message, token)
	if strings.Contains(got, token) {
		t.Errorf("токен не скрыт: %s", got)
	}
	if !strings.Contains(got, "***REDACTED***") {
		t.Errorf("ожидалась маска: %s", got)
	}
}
