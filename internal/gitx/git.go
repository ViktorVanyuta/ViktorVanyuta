// Package gitx — тонкая обёртка над git для сканера токенов: нужны только
// чтение индекса, рабочего дерева и истории.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Command выполняет git и возвращает stdout вместе с кодом возврата, чтобы
// stderr команды не попадал в вывод программы.
func Command(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "),
				strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
}

// Output выполняет git и возвращает stdout строкой.
func Output(args ...string) (string, error) {
	data, err := Command(args...)
	return string(data), err
}

// Lines выполняет git и разбивает stdout на строки. Пустой вывод даёт nil.
func Lines(args ...string) ([]string, error) {
	out, err := Output(args...)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n"), nil
}
