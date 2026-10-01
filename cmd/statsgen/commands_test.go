package main

import "testing"

func TestWriteIfChangedKeepsContent(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.svg"

	changed, err := writeIfChanged(path, "<svg/>")
	if err != nil || !changed {
		t.Fatalf("первая запись: changed=%v err=%v", changed, err)
	}
	changed, err = writeIfChanged(path, "<svg/>")
	if err != nil || changed {
		t.Fatalf("повторная запись: changed=%v err=%v", changed, err)
	}
	changed, err = writeIfChanged(path, "<svg2/>")
	if err != nil || !changed {
		t.Fatalf("изменённая запись: changed=%v err=%v", changed, err)
	}
}
