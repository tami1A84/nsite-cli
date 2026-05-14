package nip5a

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/tami1A84/nsite-cli/internal/project"
)

const KindRoot = 15128
const KindNamed = 35128

type FileAsset struct {
	Path        string `json:"path"`
	Filesystem  string `json:"filesystem"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size"`
}

type Preview struct {
	Kind   int         `json:"kind"`
	Tags   nostr.Tags  `json:"tags"`
	Assets []FileAsset `json:"assets"`
}

func Scan(root string) ([]FileAsset, error) {
	assets := []FileAsset{}
	if err := filepath.WalkDir(root, func(fp string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		f, err := os.Open(fp)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, fp)
		if err != nil {
			return err
		}
		web := "/" + filepath.ToSlash(rel)
		assets = append(assets, FileAsset{Path: web, Filesystem: fp, SHA256: hex.EncodeToString(h.Sum(nil)), Size: info.Size()})
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	return assets, nil
}

func BuildPreview(p project.Project, servers []string, rootAbs string) (*Preview, error) {
	assets, err := Scan(rootAbs)
	if err != nil {
		return nil, err
	}
	hasIndex := false
	for _, a := range assets {
		if a.Path == "/index.html" {
			hasIndex = true
			break
		}
	}
	if !hasIndex {
		return nil, fmt.Errorf("%s must contain index.html", rootAbs)
	}
	kind := KindRoot
	tags := nostr.Tags{}
	if p.Type == "named" {
		kind = KindNamed
		tags = append(tags, nostr.Tag{"d", p.D})
	}
	for _, a := range assets {
		tags = append(tags, nostr.Tag{"path", a.Path, a.SHA256})
	}
	for _, s := range servers {
		if strings.TrimSpace(s) != "" {
			tags = append(tags, nostr.Tag{"server", strings.TrimRight(s, "/")})
		}
	}
	if p.Title != "" {
		tags = append(tags, nostr.Tag{"title", p.Title})
	}
	if p.Description != "" {
		tags = append(tags, nostr.Tag{"description", p.Description})
	}
	if p.Source != "" {
		tags = append(tags, nostr.Tag{"source", p.Source})
	}
	return &Preview{Kind: kind, Tags: tags, Assets: assets}, nil
}

func WritePreview(path string, pr *Preview) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(pr, "", "  ")
	return os.WriteFile(path, append(b, '\n'), 0644)
}

func Event(pr *Preview, pubkey string) nostr.Event {
	return nostr.Event{Kind: pr.Kind, PubKey: pubkey, CreatedAt: nostr.Now(), Tags: pr.Tags, Content: ""}
}

func TagValue(tags nostr.Tags, name string) string {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == name {
			return tag[1]
		}
	}
	return ""
}

func TagValues(tags nostr.Tags, name string) []string {
	out := []string{}
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == name {
			out = append(out, tag[1])
		}
	}
	return out
}

func PathAssetsFromEvent(ev *nostr.Event) []FileAsset {
	assets := []FileAsset{}
	for _, tag := range ev.Tags {
		if len(tag) >= 3 && tag[0] == "path" {
			assets = append(assets, FileAsset{Path: tag[1], SHA256: tag[2]})
		}
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	return assets
}
