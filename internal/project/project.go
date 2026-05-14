package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Project struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	D           string `json:"d,omitempty"`
	Root        string `json:"root"`
	Source      string `json:"source,omitempty"`
}

var dtagRe = regexp.MustCompile(`^[a-z0-9-]{1,13}$`)

func Default(name, d string) Project {
	if d == "" {
		d = sanitizeD(name)
	}
	return Project{Title: name, Description: "A NIP-5A mini app", Type: "named", D: d, Root: "public"}
}

func sanitizeD(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
		if b.Len() >= 13 {
			break
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return "app"
	}
	return out
}

func Load(dir string) (*Project, error) {
	b, err := os.ReadFile(filepath.Join(dir, "nsite.json"))
	if err != nil {
		return nil, err
	}
	var p Project
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	if p.Root == "" {
		p.Root = "public"
	}
	return &p, p.Validate()
}

func (p Project) Save(dir string) error {
	b, _ := json.MarshalIndent(p, "", "  ")
	return os.WriteFile(filepath.Join(dir, "nsite.json"), append(b, '\n'), 0644)
}

func (p Project) Validate() error {
	if p.Title == "" {
		return errors.New("title is required")
	}
	if p.Type == "" {
		return errors.New("type is required")
	}
	if p.Type != "root" && p.Type != "named" {
		return errors.New("type must be root or named")
	}
	if p.Type == "named" {
		if !dtagRe.MatchString(p.D) || strings.HasSuffix(p.D, "-") {
			return fmt.Errorf("invalid d tag %q: must match ^[a-z0-9-]{1,13}$ and not end with -", p.D)
		}
	}
	if p.Type == "root" && p.D != "" {
		return errors.New("root site must not have d")
	}
	return nil
}

func Init(dir, name, d string) error {
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0755); err != nil {
		return err
	}
	p := Default(name, d)
	if err := p.Validate(); err != nil {
		return err
	}
	if err := p.Save(dir); err != nil {
		return err
	}
	index := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + p.Title + `</title>
  <link rel="stylesheet" href="/style.css">
</head>
<body>
  <main>
    <h1>` + p.Title + `</h1>
    <p>Hello from nsite-cli.</p>
  </main>
  <script src="/main.js"></script>
</body>
</html>
`
	files := map[string]string{
		"public/index.html": index,
		"public/style.css":  "body{font-family:system-ui,sans-serif;max-width:720px;margin:4rem auto;padding:0 1rem;line-height:1.6}\n",
		"public/main.js":    "console.log('hello nsite');\n",
		"public/404.html":   "<!doctype html><meta charset=\"utf-8\"><title>404</title><h1>404</h1>\n",
	}
	for rel, content := range files {
		fp := filepath.Join(dir, rel)
		if _, err := os.Stat(fp); err == nil {
			continue
		}
		if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}
