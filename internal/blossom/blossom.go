package blossom

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/tami1A84/nsite-cli/internal/nip5a"
)

func BlobURL(server, sha256 string) string {
	return strings.TrimRight(server, "/") + "/" + sha256
}

func UploadAll(ctx context.Context, servers []string, assets []nip5a.FileAsset, sk string) error {
	for _, server := range servers {
		server = strings.TrimRight(server, "/")
		if server == "" {
			continue
		}
		for _, asset := range assets {
			if err := Upload(ctx, server, asset, sk); err != nil {
				return fmt.Errorf("upload %s to %s: %w", asset.Path, server, err)
			}
		}
	}
	return nil
}

func Upload(ctx context.Context, server string, asset nip5a.FileAsset, sk string) error {
	b, err := os.ReadFile(asset.Filesystem)
	if err != nil {
		return err
	}
	pub, err := nostr.GetPublicKey(sk)
	if err != nil {
		return err
	}
	auth := nostr.Event{
		Kind:      24242,
		PubKey:    pub,
		CreatedAt: nostr.Now(),
		Content:   "Upload " + asset.SHA256,
		Tags: nostr.Tags{
			{"t", "upload"},
			{"x", asset.SHA256},
			{"expiration", fmt.Sprint(time.Now().Add(10 * time.Minute).Unix())},
		},
	}
	if err := auth.Sign(sk); err != nil {
		return err
	}
	jb, _ := json.Marshal(auth)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(server, "/")+"/upload", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Nostr "+base64.StdEncoding.EncodeToString(jb))
	req.Header.Set("Content-Type", http.DetectContentType(b))
	req.ContentLength = int64(len(b))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	return fmt.Errorf("status %s: %s", res.Status, strings.TrimSpace(string(body)))
}

func Check(ctx context.Context, server string, asset nip5a.FileAsset) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, BlobURL(server, asset.SHA256), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err == nil {
		defer res.Body.Close()
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return nil
		}
		// Some Blossom servers may not support HEAD. Fall through to GET.
		if res.StatusCode != http.StatusMethodNotAllowed && res.StatusCode != http.StatusNotImplemented && res.StatusCode != http.StatusForbidden {
			return fmt.Errorf("status %s", res.Status)
		}
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, BlobURL(server, asset.SHA256), nil)
	if err != nil {
		return err
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	return fmt.Errorf("status %s: %s", res.Status, strings.TrimSpace(string(body)))
}
