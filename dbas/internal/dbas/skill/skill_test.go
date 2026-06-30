package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSkillWithFrontMatter(t *testing.T) {
	content := `---
name: test-skill
version: 1.0.0
description: A test skill for unit testing
---
## Instructions
This is the skill content.
Do something useful.`

	sk, err := parseSkill(content)
	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}
	if sk.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got %q", sk.Name)
	}
	if sk.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", sk.Version)
	}
	if sk.Description != "A test skill for unit testing" {
		t.Errorf("expected description 'A test skill for unit testing', got %q", sk.Description)
	}
	if sk.Instructions != "## Instructions\nThis is the skill content.\nDo something useful." {
		t.Errorf("unexpected instructions:\n%q", sk.Instructions)
	}
}

func TestParseSkillNoFrontMatter(t *testing.T) {
	content := "Just some markdown content\nwithout front matter."
	sk, err := parseSkill(content)
	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}
	if sk.Name != "unknown" {
		t.Errorf("expected name 'unknown', got %q", sk.Name)
	}
	if sk.Instructions != content {
		t.Errorf("expected instructions to match input")
	}
}

func TestParseSkillMissingName(t *testing.T) {
	content := `---
version: 1.0.0
---
Content without name`
	_, err := parseSkill(content)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestParseSkillBOM(t *testing.T) {
	content := "\ufeff---\nname: bom-skill\n---\nContent with BOM"
	sk, err := parseSkill(content)
	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}
	if sk.Name != "bom-skill" {
		t.Errorf("expected name 'bom-skill', got %q", sk.Name)
	}
}

func TestParseSkillInvalidFrontMatter(t *testing.T) {
	content := "---\nno closing delimiter"
	_, err := parseSkill(content)
	if err == nil {
		t.Fatal("expected error for invalid front matter, got nil")
	}
}

func TestLoadOne(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "test.skill.md")
	content := `---
name: loaded-skill
description: Loaded from file
---
## Instructions`
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	sk, err := LoadOne(skillPath)
	if err != nil {
		t.Fatalf("LoadOne failed: %v", err)
	}
	if sk.Name != "loaded-skill" {
		t.Errorf("expected name 'loaded-skill', got %q", sk.Name)
	}
}

func TestLoadOneFileNotFound(t *testing.T) {
	_, err := LoadOne("/nonexistent/path.skill.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestLoaderLoadAll(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Create skill in dir1
	os.WriteFile(filepath.Join(dir1, "skill-a.md"), []byte("---\nname: skill-a\n---\nContent A"), 0644)
	// Create skill in dir2
	os.WriteFile(filepath.Join(dir2, "skill-b.skill.md"), []byte("---\nname: skill-b\n---\nContent B"), 0644)
	// Create a non-skill file (should be skipped)
	os.WriteFile(filepath.Join(dir1, "README.txt"), []byte("not a skill"), 0644)

	loader := NewLoader(dir1, dir2)
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	if !names["skill-a"] {
		t.Error("expected skill-a to be loaded")
	}
	if !names["skill-b"] {
		t.Error("expected skill-b to be loaded")
	}
}

func TestLoaderNonexistentDir(t *testing.T) {
	loader := NewLoader("/nonexistent/dir")
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll should not error on missing dir: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills from nonexistent dir, got %d", len(skills))
	}
}

func TestLoaderDeduplication(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "dup.skill.md"), []byte("---\nname: dup-skill\n---\nContent"), 0644)

	loader := NewLoader(dir, dir) // same dir twice
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("expected 1 skill (deduped), got %d", len(skills))
	}
}

func TestSkillToPromptFragment(t *testing.T) {
	sk := Skill{
		Name:         "test",
		Version:      "1.0.0",
		Description:  "A test skill",
		Instructions: "Do the thing\nStep by step",
	}
	prompt := sk.ToPromptFragment()
	if len(prompt) == 0 {
		t.Fatal("expected non-empty prompt fragment")
	}
}

func TestSkillJSON(t *testing.T) {
	sk := Skill{Name: "json-skill", Version: "2.0", Description: "JSON test", Instructions: "test"}
	data, err := sk.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}
