package gitx

import (
	"os/exec"
	"strings"
	"testing"
)

func TestOutputAndLines(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Skipf("git недоступен: %v (%s)", err, out)
	}
	t.Chdir(dir)

	out, err := Output("symbolic-ref", "HEAD")
	if err != nil {
		t.Fatalf("symbolic-ref: %v", err)
	}
	if !strings.HasPrefix(out, "refs/heads/") {
		t.Errorf("HEAD = %q, ожидалась ссылка на ветку", out)
	}

	lines, err := Lines("symbolic-ref", "HEAD")
	if err != nil {
		t.Fatalf("Lines: %v", err)
	}
	if len(lines) != 1 || strings.TrimSpace(lines[0]) == "" {
		t.Errorf("Lines вернул %q, ожидалась одна непустая строка", lines)
	}
}

func TestLinesEmptyOutputIsNil(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Skipf("git недоступен: %v (%s)", err, out)
	}
	t.Chdir(dir)

	lines, err := Lines("for-each-ref", "--format=%(refname)")
	if err != nil {
		t.Fatalf("for-each-ref: %v", err)
	}
	if lines != nil {
		t.Errorf("пустой вывод дал %q, ожидался nil", lines)
	}
}

func TestCommandReportsError(t *testing.T) {
	if _, err := Command("выдуманная-подкоманда"); err == nil {
		t.Fatal("ожидалась ошибка для несуществующей команды git")
	}
}
