package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v2"
	"github.com/tami1A84/nsite-cli/internal/blossom"
	"github.com/tami1A84/nsite-cli/internal/config"
	"github.com/tami1A84/nsite-cli/internal/nip5a"
	"github.com/tami1A84/nsite-cli/internal/project"
)

const version = "0.1.0"

func main() {
	app := &cli.App{
		Name:    "nsite-cli",
		Usage:   "create, test, and publish NIP-5A static websites",
		Version: version,
		Commands: []*cli.Command{
			cmdInit(), cmdDev(), cmdBuild(), cmdPublish(), cmdInspect(), cmdDoctor(), cmdConfig(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		color.Red("error: %v", err)
		os.Exit(1)
	}
}

func cmdInit() *cli.Command {
	return &cli.Command{Name: "init", Usage: "create a new nsite project", Flags: []cli.Flag{&cli.StringFlag{Name: "d", Usage: "named site d tag"}, &cli.StringFlag{Name: "title", Usage: "site title"}}, Action: func(c *cli.Context) error {
		dir := c.Args().First()
		if dir == "" {
			dir = "my-nsite"
		}
		// urfave/cli stops parsing command flags after the first positional arg.
		// Accept both `nsite-cli init --d foo app` and `nsite-cli init app --d foo`.
		d := c.String("d")
		if d == "" {
			d = stringFlagAnywhere("d")
		}
		name := c.String("title")
		if name == "" {
			name = stringFlagAnywhere("title")
		}
		if name == "" {
			name = filepath.Base(filepath.Clean(dir))
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := project.Init(dir, name, d); err != nil {
			return err
		}
		color.Green("created %s", dir)
		fmt.Println("next:")
		fmt.Printf("  cd %s\n  nsite-cli dev\n  nsite-cli build\n  nsite-cli publish\n", dir)
		return nil
	}}
}

func stringFlagAnywhere(name string) string {
	long := "--" + name
	prefix := long + "="
	for i, arg := range os.Args {
		if arg == long && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

func cmdDev() *cli.Command {
	return &cli.Command{Name: "dev", Usage: "serve locally", Flags: []cli.Flag{&cli.StringFlag{Name: "addr", Value: ":3128", Usage: "listen address"}}, Action: func(c *cli.Context) error {
		p, err := project.Load(".")
		if err != nil {
			return err
		}
		root := filepath.Clean(p.Root)
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Clean("/" + r.URL.Path)
			if strings.HasSuffix(r.URL.Path, "/") {
				path = filepath.Join(path, "index.html")
			}
			fp := filepath.Join(root, path)
			if info, err := os.Stat(fp); err == nil && !info.IsDir() {
				http.ServeFile(w, r, fp)
				return
			}
			if info, err := os.Stat(filepath.Join(root, "404.html")); err == nil && !info.IsDir() {
				b, err := os.ReadFile(filepath.Join(root, "404.html"))
				if err == nil {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(404)
					_, _ = w.Write(b)
					return
				}
			}
			http.NotFound(w, r)
		})
		color.Green("serving %s at http://localhost%s", root, c.String("addr"))
		return http.ListenAndServe(c.String("addr"), h)
	}}
}

func buildPreview() (*project.Project, *nip5a.Preview, error) {
	p, err := project.Load(".")
	if err != nil {
		return nil, nil, err
	}
	cfg, _ := config.Load()
	servers := []string{}
	if cfg != nil {
		servers = cfg.Blossom.Servers
	}
	root, err := filepath.Abs(p.Root)
	if err != nil {
		return nil, nil, err
	}
	pr, err := nip5a.BuildPreview(*p, servers, root)
	if err != nil {
		return nil, nil, err
	}
	return p, pr, nil
}

func cmdBuild() *cli.Command {
	return &cli.Command{Name: "build", Usage: "generate NIP-5A manifest preview", Action: func(c *cli.Context) error {
		_, pr, err := buildPreview()
		if err != nil {
			return err
		}
		if err := nip5a.WritePreview(filepath.Join(".nsite", "manifest-preview.json"), pr); err != nil {
			return err
		}
		color.Green("manifest preview written: .nsite/manifest-preview.json")
		fmt.Printf("kind: %d\nfiles: %d\n", pr.Kind, len(pr.Assets))
		return nil
	}}
}

func cmdPublish() *cli.Command {
	return &cli.Command{Name: "publish", Usage: "upload files to Blossom and publish manifest event", Flags: []cli.Flag{&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "skip confirmation"}, &cli.StringFlag{Name: "host", Usage: "nsite host domain for canonical URL display"}}, Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		sk, err := cfg.SecretHex()
		if err != nil {
			return err
		}
		pub, err := nostr.GetPublicKey(sk)
		if err != nil {
			return err
		}
		p, pr, err := buildPreview()
		if err != nil {
			return err
		}
		if len(cfg.Blossom.Servers) == 0 {
			return fmt.Errorf("no blossom.servers configured")
		}
		if len(cfg.WriteRelays()) == 0 {
			return fmt.Errorf("no write relays configured")
		}
		fmt.Printf("title: %s\nkind: %d\nfiles: %d\nblossom: %v\nrelays: %v\n", p.Title, pr.Kind, len(pr.Assets), cfg.Blossom.Servers, cfg.WriteRelays())
		if !c.Bool("yes") {
			fmt.Print("publish? [y/N] ")
			var ans string
			fmt.Scanln(&ans)
			if strings.ToLower(ans) != "y" && strings.ToLower(ans) != "yes" {
				return nil
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		color.Cyan("uploading assets...")
		if err := blossom.UploadAll(ctx, cfg.Blossom.Servers, pr.Assets, sk); err != nil {
			return err
		}
		ev := nip5a.Event(pr, pub)
		if err := ev.Sign(sk); err != nil {
			return err
		}
		pool := nostr.NewSimplePool(ctx)
		color.Cyan("publishing event %s...", ev.ID)
		for res := range pool.PublishMany(ctx, cfg.WriteRelays(), ev) {
			if res.Error != nil {
				color.Red("%s: %v", res.Relay, res.Error)
			} else {
				color.Green("%s: ok", res.Relay)
			}
		}
		nevent, _ := nip19.EncodeEvent(ev.ID, cfg.WriteRelays(), pub)
		color.Green("published")
		fmt.Println("event:", ev.ID)
		if nevent != "" {
			fmt.Println("nevent:", nevent)
		}
		printCanonicalHint(p, pub, firstNonEmpty(c.String("host"), stringFlagAnywhere("host"), cfg.Nsite.Host))
		printAssets(pr, cfg.Blossom.Servers)
		return nil
	}}
}

func printAssets(pr *nip5a.Preview, servers []string) {
	fmt.Println("assets:")
	if len(servers) == 0 {
		for _, a := range pr.Assets {
			fmt.Printf("  %s  %s\n", a.Path, a.SHA256)
		}
		return
	}
	for _, a := range pr.Assets {
		fmt.Printf("  %s\n", a.Path)
		for _, s := range servers {
			fmt.Printf("    %s\n", blossom.BlobURL(s, a.SHA256))
		}
	}
}

func printCanonicalHint(p *project.Project, pub string, host string) {
	fmt.Println("nsite:")
	fmt.Printf("  pubkey: %s\n", pub)
	if p.Type == "root" {
		npub, _ := nip19.EncodePublicKey(pub)
		if npub != "" {
			fmt.Printf("  root host label: %s\n", npub)
			if host = cleanHost(host); host != "" {
				fmt.Printf("  url: https://%s.%s/\n", npub, host)
			}
		}
		return
	}
	b36, err := pubkeyBase36(pub)
	fmt.Printf("  kind: %d\n", nip5a.KindNamed)
	fmt.Printf("  d: %s\n", p.D)
	if err != nil {
		color.Yellow("  pubkeyB36: %v", err)
		return
	}
	label := b36 + p.D
	fmt.Printf("  pubkeyB36: %s\n", b36)
	fmt.Printf("  host label: %s\n", label)
	if host = cleanHost(host); host != "" {
		fmt.Printf("  url: https://%s.%s/\n", label, host)
	} else {
		fmt.Println("  url: set config nsite.host or pass --host to display full canonical URL")
	}
}

func pubkeyBase36(hexPub string) (string, error) {
	b, err := hex.DecodeString(hexPub)
	if err != nil {
		return "", err
	}
	if len(b) != 32 {
		return "", fmt.Errorf("pubkey must be 32 bytes, got %d", len(b))
	}
	n := new(big.Int).SetBytes(b)
	s := strings.ToLower(n.Text(36))
	if len(s) > 50 {
		return "", fmt.Errorf("base36 pubkey too long: %d", len(s))
	}
	return strings.Repeat("0", 50-len(s)) + s, nil
}

func pubkeyBase36Decode(s string) (string, error) {
	if len(s) != 50 {
		return "", fmt.Errorf("pubkeyB36 must be 50 chars, got %d", len(s))
	}
	n, ok := new(big.Int).SetString(s, 36)
	if !ok {
		return "", fmt.Errorf("invalid base36 pubkey")
	}
	b := n.Bytes()
	if len(b) > 32 {
		return "", fmt.Errorf("decoded pubkey is too large")
	}
	padded := append(make([]byte, 32-len(b)), b...)
	return hex.EncodeToString(padded), nil
}

func cleanHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	return strings.Trim(host, "/")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func cmdInspect() *cli.Command {
	return &cli.Command{Name: "inspect", Usage: "inspect a NIP-5A event by event id/nevent", Flags: []cli.Flag{&cli.BoolFlag{Name: "json", Usage: "print raw event json"}, &cli.StringFlag{Name: "host", Usage: "nsite host domain for canonical URL display"}}, Action: func(c *cli.Context) error {
		id := c.Args().First()
		if id == "" {
			return fmt.Errorf("usage: nsite-cli inspect <event-id|nevent>")
		}
		eventID := id
		if strings.HasPrefix(id, "nevent") {
			prefix, v, err := nip19.Decode(id)
			if err != nil {
				return err
			}
			if prefix != "nevent" {
				return fmt.Errorf("expected nevent, got %s", prefix)
			}
			if data, ok := v.(nostr.EventPointer); ok {
				eventID = data.ID
			} else {
				b, _ := json.Marshal(v)
				var m map[string]any
				_ = json.Unmarshal(b, &m)
				if s, ok := m["id"].(string); ok {
					eventID = s
				}
			}
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		relays := cfg.ReadRelays()
		if len(relays) == 0 {
			relays = cfg.WriteRelays()
		}
		if len(relays) == 0 {
			return fmt.Errorf("no read relays configured")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pool := nostr.NewSimplePool(ctx)
		res := pool.QuerySingle(ctx, relays, nostr.Filter{IDs: []string{eventID}, Limit: 1})
		if res == nil || res.Event == nil {
			return fmt.Errorf("event not found: %s", eventID)
		}
		ev := res.Event
		if c.Bool("json") {
			b, _ := json.MarshalIndent(ev, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		fmt.Printf("event: %s\nrelay: %s\nkind: %d\nauthor: %s\ncreated_at: %d\n", ev.ID, res.Relay.URL, ev.Kind, ev.PubKey, ev.CreatedAt)
		dtag := nip5a.TagValue(ev.Tags, "d")
		fmt.Printf("title: %s\ndescription: %s\nd: %s\n", nip5a.TagValue(ev.Tags, "title"), nip5a.TagValue(ev.Tags, "description"), dtag)
		printEventCanonicalHint(ev, dtag, firstNonEmpty(c.String("host"), stringFlagAnywhere("host"), cfg.Nsite.Host))
		servers := nip5a.TagValues(ev.Tags, "server")
		fmt.Printf("servers: %v\n", servers)
		assets := nip5a.PathAssetsFromEvent(ev)
		fmt.Println("paths:")
		for _, a := range assets {
			fmt.Printf("  %s  %s\n", a.Path, a.SHA256)
			for _, s := range servers {
				fmt.Printf("    %s\n", blossom.BlobURL(s, a.SHA256))
			}
		}
		return nil
	}}
}

func printEventCanonicalHint(ev *nostr.Event, dtag string, host string) {
	fmt.Println("canonical:")
	if ev.Kind == nip5a.KindRoot {
		npub, _ := nip19.EncodePublicKey(ev.PubKey)
		if npub != "" {
			fmt.Printf("  root host label: %s\n", npub)
			if host = cleanHost(host); host != "" {
				fmt.Printf("  url: https://%s.%s/\n", npub, host)
			}
		}
		return
	}
	if ev.Kind != nip5a.KindNamed {
		fmt.Println("  not a NIP-5A root/named site event")
		return
	}
	b36, err := pubkeyBase36(ev.PubKey)
	if err != nil {
		color.Yellow("  pubkeyB36: %v", err)
		return
	}
	label := b36 + dtag
	fmt.Printf("  pubkeyB36: %s\n", b36)
	fmt.Printf("  host label: %s\n", label)
	if host = cleanHost(host); host != "" {
		fmt.Printf("  url: https://%s.%s/\n", label, host)
	}
}

func cmdDoctor() *cli.Command {
	return &cli.Command{Name: "doctor", Usage: "check config, project, relays, and Blossom availability", Flags: []cli.Flag{&cli.BoolFlag{Name: "online", Value: true, Usage: "check relay and Blossom network access"}}, Action: func(c *cli.Context) error {
		fp, _ := config.ConfigPath()
		fmt.Println("config:", fp)
		cfg, err := config.Load()
		if err != nil {
			color.Yellow("config not loaded: %v", err)
			created, e := config.EnsureExample()
			if e == nil {
				color.Yellow("example created: %s", created)
			}
			return err
		}
		if _, err := cfg.SecretHex(); err != nil {
			color.Red("privatekey: %v", err)
		} else {
			color.Green("privatekey: ok")
		}
		fmt.Printf("read relays: %v\n", cfg.ReadRelays())
		fmt.Printf("write relays: %v\n", cfg.WriteRelays())
		fmt.Printf("blossom servers: %v\n", cfg.Blossom.Servers)
		var pr *nip5a.Preview
		if p, err := project.Load("."); err == nil {
			color.Green("project: %s (%s)", p.Title, p.Type)
			root, _ := filepath.Abs(p.Root)
			if built, err := nip5a.BuildPreview(*p, cfg.Blossom.Servers, root); err == nil {
				pr = built
				color.Green("manifest: ok (%d files)", len(pr.Assets))
			} else {
				color.Red("manifest: %v", err)
			}
		} else {
			color.Yellow("project: %v", err)
		}
		if c.Bool("online") {
			checkOnline(cfg, pr)
		}
		return nil
	}}
}

func checkOnline(cfg *config.Config, pr *nip5a.Preview) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if len(cfg.WriteRelays()) > 0 {
		pool := nostr.NewSimplePool(ctx)
		dummy := nostr.Event{Kind: 1, Content: "", CreatedAt: nostr.Now()}
		for _, r := range cfg.WriteRelays() {
			if relay, err := pool.EnsureRelay(r); err == nil && relay != nil {
				_ = relay
				color.Green("relay: %s ok", r)
			} else {
				color.Red("relay: %s %v", r, err)
			}
		}
		_ = dummy
	}
	if pr != nil && len(pr.Assets) > 0 {
		for _, server := range cfg.Blossom.Servers {
			server = strings.TrimRight(server, "/")
			for _, asset := range pr.Assets {
				if err := blossom.Check(ctx, server, asset); err != nil {
					color.Yellow("blossom: %s %s not found yet (%v)", server, asset.Path, err)
				} else {
					color.Green("blossom: %s %s ok", server, asset.Path)
				}
			}
		}
	}
}

func cmdConfig() *cli.Command {
	return &cli.Command{Name: "config", Usage: "config helpers", Subcommands: []*cli.Command{
		{Name: "path", Usage: "print config path", Action: func(c *cli.Context) error {
			fp, err := config.ConfigPath()
			if err != nil {
				return err
			}
			fmt.Println(fp)
			return nil
		}},
		{Name: "init", Usage: "create example config", Action: func(c *cli.Context) error {
			fp, err := config.EnsureExample()
			if err != nil {
				return err
			}
			fmt.Println(fp)
			return nil
		}},
		{Name: "get", Usage: "get a config value", Action: func(c *cli.Context) error {
			key := c.Args().First()
			if key == "" {
				return fmt.Errorf("usage: nsite-cli config get <key>")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			switch key {
			case "nsite.host":
				fmt.Println(cfg.Nsite.Host)
			case "privatekey":
				fmt.Println(cfg.PrivateKey)
			case "blossom.servers":
				for _, s := range cfg.Blossom.Servers {
					fmt.Println(s)
				}
			default:
				return fmt.Errorf("unsupported config key %q", key)
			}
			return nil
		}},
		{Name: "set", Usage: "set a config value", Action: func(c *cli.Context) error {
			key := c.Args().Get(0)
			value := c.Args().Get(1)
			if key == "" || value == "" {
				return fmt.Errorf("usage: nsite-cli config set <key> <value>")
			}
			cfg, err := config.Load()
			if err != nil {
				if _, e := config.EnsureExample(); e != nil {
					return err
				}
				cfg, err = config.Load()
				if err != nil {
					return err
				}
			}
			switch key {
			case "nsite.host":
				cfg.Nsite.Host = cleanHost(value)
			case "privatekey":
				cfg.PrivateKey = value
			case "blossom.servers":
				cfg.Blossom.Servers = splitCSV(value)
				cfg.BlossomServers = nil
			default:
				return fmt.Errorf("unsupported config key %q", key)
			}
			fp, err := config.Save(cfg)
			if err != nil {
				return err
			}
			color.Green("updated %s", fp)
			return nil
		}},
		{Name: "unset", Usage: "unset a config value", Action: func(c *cli.Context) error {
			key := c.Args().First()
			if key == "" {
				return fmt.Errorf("usage: nsite-cli config unset <key>")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			switch key {
			case "nsite.host":
				cfg.Nsite.Host = ""
			default:
				return fmt.Errorf("unsupported config key %q", key)
			}
			fp, err := config.Save(cfg)
			if err != nil {
				return err
			}
			color.Green("updated %s", fp)
			return nil
		}},
	}}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
