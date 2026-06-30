package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Description  string   `json:"description,omitempty"`
	Tools        []string `json:"tools,omitempty"`
	Instructions string   `json:"instructions"`
}

type Loader struct {
	dirs []string
}

func NewLoader(dirs ...string) *Loader {
	return &Loader{dirs: dirs}
}

func (l *Loader) LoadAll() ([]Skill, error) {
	var skills []Skill
	seen := make(map[string]bool)
	for _, dir := range l.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || (!strings.HasSuffix(e.Name(), ".md") && !strings.HasSuffix(e.Name(), ".skill.md")) {
				continue
			}
			if seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			sk, err := parseSkill(string(data))
			if err != nil {
				continue
			}
			if sk.Name != "" {
				skills = append(skills, sk)
			}
		}
	}
	return skills, nil
}

func LoadOne(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sk, err := parseSkill(string(data))
	if err != nil {
		return nil, err
	}
	return &sk, nil
}

func parseSkill(content string) (Skill, error) {
	content = strings.TrimSpace(content)
	// Strip UTF-8 BOM if present
	content = strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(content, "---") {
		return Skill{
			Name:         "unknown",
			Instructions: content,
		}, nil
	}
	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		return Skill{}, fmt.Errorf("invalid skill front matter")
	}
	var sk Skill
	for _, line := range strings.Split(strings.TrimSpace(parts[0]), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "name:"):
			sk.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		case strings.HasPrefix(line, "version:"):
			sk.Version = strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		case strings.HasPrefix(line, "description:"):
			sk.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	sk.Instructions = strings.TrimSpace(parts[1])
	if sk.Name == "" {
		return Skill{}, fmt.Errorf("skill name is required")
	}
	return sk, nil
}

func (s *Skill) ToPromptFragment() string {
	var b strings.Builder
	b.WriteString("=== Skill: ")
	b.WriteString(s.Name)
	b.WriteString(" ===\n")
	if s.Description != "" {
		b.WriteString(s.Description)
		b.WriteString("\n\n")
	}
	b.WriteString(s.Instructions)
	return b.String()
}

func (s Skill) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"name":         s.Name,
		"version":      s.Version,
		"description":  s.Description,
		"instructions": s.Instructions,
	})
}


