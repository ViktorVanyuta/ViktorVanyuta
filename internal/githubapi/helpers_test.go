package githubapi

import (
	"io"
	"os"
	"strings"
	"sync"
)

// captureStderr подменяет os.Stderr на буфер. Возвращаемая функция восстанавливает
// поток и наполняет буфер; её безопасно вызывать несколько раз, поэтому в тестах
// можно звать и defer restore(), и явный restore() перед проверкой.
func captureStderr(buffer *strings.Builder) func() {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	original := os.Stderr
	os.Stderr = writer

	collected := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(reader)
		collected <- string(data)
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			writer.Close()
			os.Stderr = original
			buffer.WriteString(<-collected)
			reader.Close()
		})
	}
}
