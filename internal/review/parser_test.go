package review

import (
	"strings"
	"testing"
)

func TestParseDiff_Empty(t *testing.T) {
	info := ParseDiff("")

	if info.FilesChanged != 0 {
		t.Errorf("expected 0 files changed, got %d", info.FilesChanged)
	}
	if info.Additions != 0 {
		t.Errorf("expected 0 additions, got %d", info.Additions)
	}
	if info.Deletions != 0 {
		t.Errorf("expected 0 deletions, got %d", info.Deletions)
	}
}

func TestParseDiff_SingleFile(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

 func main() {
+	fmt.Println("Hello")
 }
`
	info := ParseDiff(diff)

	if info.FilesChanged != 1 {
		t.Errorf("expected 1 file changed, got %d", info.FilesChanged)
	}
	if info.Additions != 1 {
		t.Errorf("expected 1 addition, got %d", info.Additions)
	}
	if info.Deletions != 0 {
		t.Errorf("expected 0 deletions, got %d", info.Deletions)
	}

	if len(info.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(info.Files))
	}

	file := info.Files[0]
	if file.Path != "main.go" {
		t.Errorf("expected path 'main.go', got '%s'", file.Path)
	}
	if file.Status != "modified" {
		t.Errorf("expected status 'modified', got '%s'", file.Status)
	}
	if file.Language != "go" {
		t.Errorf("expected language 'go', got '%s'", file.Language)
	}
}

func TestParseDiff_MultipleFiles(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
 func main() {}
diff --git a/utils.go b/utils.go
--- a/utils.go
+++ b/utils.go
@@ -1,4 +1,3 @@
 package main
-func old() {}
 func helper() {}
`
	info := ParseDiff(diff)

	if info.FilesChanged != 2 {
		t.Errorf("expected 2 files changed, got %d", info.FilesChanged)
	}
	if info.Additions != 1 {
		t.Errorf("expected 1 addition, got %d", info.Additions)
	}
	if info.Deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", info.Deletions)
	}
}

func TestParseDiff_NewFile(t *testing.T) {
	diff := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..abc123
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+package main
+
+func newFunc() {}
`
	info := ParseDiff(diff)

	if len(info.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(info.Files))
	}

	file := info.Files[0]
	if file.Status != "added" {
		t.Errorf("expected status 'added', got '%s'", file.Status)
	}
	if info.Additions != 3 {
		t.Errorf("expected 3 additions, got %d", info.Additions)
	}
}

func TestParseDiff_DeletedFile(t *testing.T) {
	diff := `diff --git a/old.go b/old.go
deleted file mode 100644
index abc123..0000000
--- a/old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func oldFunc() {}
`
	info := ParseDiff(diff)

	if len(info.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(info.Files))
	}

	file := info.Files[0]
	if file.Status != "deleted" {
		t.Errorf("expected status 'deleted', got '%s'", file.Status)
	}
	if info.Deletions != 3 {
		t.Errorf("expected 3 deletions, got %d", info.Deletions)
	}
}

func TestParseDiff_RenamedFile(t *testing.T) {
	diff := `diff --git a/old_name.go b/new_name.go
similarity index 100%
rename from old_name.go
rename to new_name.go
`
	info := ParseDiff(diff)

	if len(info.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(info.Files))
	}

	file := info.Files[0]
	if file.Status != "renamed" {
		t.Errorf("expected status 'renamed', got '%s'", file.Status)
	}
	if file.OldPath != "old_name.go" {
		t.Errorf("expected old path 'old_name.go', got '%s'", file.OldPath)
	}
	if file.Path != "new_name.go" {
		t.Errorf("expected new path 'new_name.go', got '%s'", file.Path)
	}
}

func TestParseDiff_BinaryFile(t *testing.T) {
	diff := `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ
`
	info := ParseDiff(diff)

	if len(info.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(info.Files))
	}

	file := info.Files[0]
	if !file.IsBinary {
		t.Error("expected file to be marked as binary")
	}
}

func TestParseDiff_Hunks(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

 func main() {
+	fmt.Println("Hello")
 }
@@ -10,3 +11,5 @@ func other() {
 	return
+	// new line
+	// another new line
 }
`
	info := ParseDiff(diff)

	if len(info.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(info.Files))
	}

	file := info.Files[0]
	if len(file.Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(file.Hunks))
	}

	hunk1 := file.Hunks[0]
	if hunk1.OldStart != 1 {
		t.Errorf("expected hunk1 old start 1, got %d", hunk1.OldStart)
	}
	if hunk1.NewStart != 1 {
		t.Errorf("expected hunk1 new start 1, got %d", hunk1.NewStart)
	}

	hunk2 := file.Hunks[1]
	if hunk2.OldStart != 10 {
		t.Errorf("expected hunk2 old start 10, got %d", hunk2.OldStart)
	}
	if hunk2.NewStart != 11 {
		t.Errorf("expected hunk2 new start 11, got %d", hunk2.NewStart)
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"app.ts", "typescript"},
		{"component.tsx", "typescript-react"},
		{"component.jsx", "javascript-react"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"program.c", "c"},
		{"program.cpp", "cpp"},
		{"App.cs", "csharp"},
		{"script.rb", "ruby"},
		{"page.php", "php"},
		{"app.swift", "swift"},
		{"Main.kt", "kotlin"},
		{"App.scala", "scala"},
		{"script.sh", "shell"},
		{"config.yaml", "yaml"},
		{"data.json", "json"},
		{"page.html", "html"},
		{"style.css", "css"},
		{"style.scss", "scss"},
		{"query.sql", "sql"},
		{"README.md", "markdown"},
		{"Dockerfile", "dockerfile"},
		{"Makefile", "makefile"},
		{".gitignore", "ignore"},
		{"unknown.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := detectLanguage(tt.path)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetDiffSummary(t *testing.T) {
	info := &DiffInfo{
		FilesChanged: 3,
		Additions:    50,
		Deletions:    20,
		Files: []FileDiff{
			{Path: "main.go", Status: "modified", Language: "go"},
			{Path: "README.md", Status: "modified", Language: "markdown"},
			{Path: "new.py", Status: "added", Language: "python"},
		},
	}

	summary := GetDiffSummary(info)

	if !strings.Contains(summary, "Files changed: 3") {
		t.Error("summary should contain files changed count")
	}
	if !strings.Contains(summary, "Additions: 50") {
		t.Error("summary should contain additions count")
	}
	if !strings.Contains(summary, "Deletions: 20") {
		t.Error("summary should contain deletions count")
	}
	if !strings.Contains(summary, "main.go") {
		t.Error("summary should list main.go")
	}
	if !strings.Contains(summary, "[go]") {
		t.Error("summary should show language for main.go")
	}
}

func TestParseDiff_LineNumbers(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -5,4 +5,5 @@ package main
 func main() {
 	existing()
+	newLine()
 }
`
	info := ParseDiff(diff)

	if len(info.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(info.Files))
	}

	file := info.Files[0]
	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}

	hunk := file.Hunks[0]

	// Find the addition line
	var addedLine *DiffLine
	for i := range hunk.Lines {
		if hunk.Lines[i].Type == LineAddition {
			addedLine = &hunk.Lines[i]
			break
		}
	}

	if addedLine == nil {
		t.Fatal("expected to find an addition line")
	}

	// The addition is at new line 7 (starts at 5, 2 context lines, then addition)
	if addedLine.NewLine != 7 {
		t.Errorf("expected new line 7, got %d", addedLine.NewLine)
	}
}
