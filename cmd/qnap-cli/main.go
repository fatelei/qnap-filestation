package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/pkg/filestation"
)

// These are set by GoReleaser via -ldflags "-X main.version=... -X main.commit=... -X main.date=..."
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type globalConfig struct {
	host        string
	port        int
	user        string
	pass        string
	insecure    bool
	timeout     time.Duration
	logJSON     bool
	output      string // json|text
	showVersion bool
}

func parseGlobalFlags(args []string) (*globalConfig, []string) {
	fs := flag.NewFlagSet("qnap-cli", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	cfg := &globalConfig{}

	defaultHost := os.Getenv("QNAP_HOST")
	defaultUser := os.Getenv("QNAP_USER")
	defaultPass := os.Getenv("QNAP_PASS")
	defaultPort := 0
	if p := os.Getenv("QNAP_PORT"); p != "" {
		var pi int
		_, _ = fmt.Sscanf(p, "%d", &pi)
		defaultPort = pi
	}
	insecureEnv := strings.ToLower(os.Getenv("QNAP_INSECURE"))
	defaultInsecure := insecureEnv == "1" || insecureEnv == "true" || insecureEnv == "yes"

	fs.StringVar(&cfg.host, "host", defaultHost, "QNAP host (env: QNAP_HOST)")
	fs.IntVar(&cfg.port, "port", defaultPort, "QNAP port (env: QNAP_PORT); 0 = auto by scheme")
	fs.StringVar(&cfg.user, "user", defaultUser, "QNAP username (env: QNAP_USER)")
	fs.StringVar(&cfg.pass, "pass", defaultPass, "QNAP password (env: QNAP_PASS)")
	fs.BoolVar(&cfg.insecure, "insecure", defaultInsecure, "Use HTTP / skip TLS verify (env: QNAP_INSECURE)")
	fs.DurationVar(&cfg.timeout, "timeout", 30*time.Second, "HTTP timeout (e.g. 30s, 2m)")
	fs.BoolVar(&cfg.logJSON, "log-json", true, "Log in JSON format")
	fs.StringVar(&cfg.output, "output", "json", "Output format: json|text")
	fs.BoolVar(&cfg.showVersion, "version", false, "Show version and exit")

	// Stop flag parsing at first non-flag to preserve subcommand args
	_ = fs.Parse(args)
	return cfg, fs.Args()
}

func newClient(cfg *globalConfig) (*api.Client, error) {
	if cfg.host == "" || cfg.user == "" || cfg.pass == "" {
		return nil, errors.New("host, user, and pass are required (use flags or env QNAP_HOST/QNAP_USER/QNAP_PASS)")
	}

	// Configure structured logger
	var handler slog.Handler
	if cfg.logJSON {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(handler)

	ac := &api.Config{
		Host:     cfg.host,
		Port:     cfg.port,
		Username: cfg.user,
		Password: cfg.pass,
		Insecure: cfg.insecure,
		Timeout:  cfg.timeout,
		Logger:   logger,
	}
	return api.NewClient(ac)
}

func withClient(fn func(ctx context.Context, fs *filestation.FileStationService) error, cfg *globalConfig) error {
	client, err := newClient(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		return err
	}
	defer func() { _ = client.Logout(ctx) }()
	service := filestation.NewFileStationService(client)
	return fn(ctx, service)
}

func printOutput(v interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json", "":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case "text":
		// Fallback: print JSON compact on one line to keep implementation simple
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func cmdLS(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	var path string
	var limit, offset int
	fs.StringVar(&path, "path", "/", "Remote path to list")
	fs.IntVar(&limit, "limit", 100, "Max entries to return")
	fs.IntVar(&offset, "offset", 0, "Pagination offset")
	_ = fs.Parse(args)

	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		items, err := svc.ListFiles(ctx, path, &filestation.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return err
		}
		return printOutput(struct {
			Path  string             `json:"path"`
			Items []filestation.File `json:"items"`
		}{Path: path, Items: items}, cfg.output)
	}, cfg)
}

func cmdStat(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("stat", flag.ExitOnError)
	var path string
	fs.StringVar(&path, "path", "", "Remote path to stat (required)")
	_ = fs.Parse(args)
	if path == "" {
		return errors.New("-path is required")
	}

	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		info, err := svc.GetFileInfo(ctx, path)
		if err != nil {
			return err
		}
		return printOutput(info, cfg.output)
	}, cfg)
}

func cmdMkdir(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("mkdir", flag.ExitOnError)
	var path string
	fs.StringVar(&path, "path", "", "Remote directory path to create (required)")
	_ = fs.Parse(args)
	if path == "" {
		return errors.New("-path is required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		if err := svc.CreateFolder(ctx, path); err != nil {
			return err
		}
		return printOutput(map[string]any{"ok": true, "path": path}, cfg.output)
	}, cfg)
}

func cmdRM(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("rm", flag.ExitOnError)
	var path string
	fs.StringVar(&path, "path", "", "Remote file/folder path to delete (required)")
	_ = fs.Parse(args)
	if path == "" {
		return errors.New("-path is required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		// Try file delete first, fall back to folder delete
		if err := svc.DeleteFile(ctx, path); err != nil {
			// attempt folder delete
			if err2 := svc.DeleteFolder(ctx, path); err2 != nil {
				return err
			}
		}
		return printOutput(map[string]any{"ok": true, "path": path}, cfg.output)
	}, cfg)
}

func cmdMV(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("mv", flag.ExitOnError)
	var src, dst string
	fs.StringVar(&src, "src", "", "Source path (required)")
	fs.StringVar(&dst, "dst", "", "Destination folder path or new name (required)")
	_ = fs.Parse(args)
	if src == "" || dst == "" {
		return errors.New("-src and -dst are required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		// Prefer move API; if fails as file rename, try rename
		if err := svc.MoveFile(ctx, src, dst, &filestation.CopyMoveOptions{}); err != nil {
			if err2 := svc.RenameFile(ctx, src, dst); err2 != nil {
				return err
			}
		}
		return printOutput(map[string]any{"ok": true, "src": src, "dst": dst}, cfg.output)
	}, cfg)
}

func cmdCP(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("cp", flag.ExitOnError)
	var src, dst string
	var overwrite bool
	fs.StringVar(&src, "src", "", "Source path (required)")
	fs.StringVar(&dst, "dst", "", "Destination folder path (required)")
	fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing")
	_ = fs.Parse(args)
	if src == "" || dst == "" {
		return errors.New("-src and -dst are required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		if err := svc.CopyFile(ctx, src, dst, &filestation.CopyMoveOptions{Overwrite: overwrite}); err != nil {
			return err
		}
		return printOutput(map[string]any{"ok": true, "src": src, "dst": dst, "overwrite": overwrite}, cfg.output)
	}, cfg)
}

func cmdUpload(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	var local, remote string
	fs.StringVar(&local, "local", "", "Local file path to upload (required)")
	fs.StringVar(&remote, "remote", "", "Remote destination folder path (required)")
	_ = fs.Parse(args)
	if local == "" || remote == "" {
		return errors.New("-local and -remote are required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		resp, err := svc.UploadFile(ctx, local, remote, &filestation.UploadOptions{Overwrite: true})
		if err != nil {
			return err
		}
		return printOutput(resp, cfg.output)
	}, cfg)
}

func cmdDownload(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	var remote, local string
	fs.StringVar(&remote, "remote", "", "Remote file path to download (required)")
	fs.StringVar(&local, "local", "", "Local destination file path (required)")
	_ = fs.Parse(args)
	if local == "" || remote == "" {
		return errors.New("-local and -remote are required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		if err := svc.DownloadFile(ctx, remote, local, nil); err != nil {
			return err
		}
		return printOutput(map[string]any{"ok": true, "remote": remote, "local": local}, cfg.output)
	}, cfg)
}

func cmdSearch(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	var path, pattern string
	fs.StringVar(&path, "path", "/", "Remote root path to search")
	fs.StringVar(&pattern, "pattern", "", "Search pattern (e.g., *.txt) (required)")
	_ = fs.Parse(args)
	if pattern == "" {
		return errors.New("-pattern is required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		items, err := svc.SearchByPattern(ctx, path, pattern)
		if err != nil {
			return err
		}
		return printOutput(struct {
			Path    string             `json:"path"`
			Pattern string             `json:"pattern"`
			Items   []filestation.File `json:"items"`
		}{Path: path, Pattern: pattern, Items: items}, cfg.output)
	}, cfg)
}

func cmdShareCreate(cfg *globalConfig, args []string) error {
	fs := flag.NewFlagSet("share-create", flag.ExitOnError)
	var path string
	fs.StringVar(&path, "path", "", "Remote file/folder path to share (required)")
	_ = fs.Parse(args)
	if path == "" {
		return errors.New("-path is required")
	}
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		link, err := svc.CreateShareLink(ctx, path, &filestation.ShareLinkOptions{Writeable: false})
		if err != nil {
			return err
		}
		return printOutput(link, cfg.output)
	}, cfg)
}

func cmdSharesList(cfg *globalConfig, args []string) error {
	return withClient(func(ctx context.Context, svc *filestation.FileStationService) error {
		links, err := svc.GetShareList(ctx)
		if err != nil {
			return err
		}
		return printOutput(links, cfg.output)
	}, cfg)
}

func runCommand(cfg *globalConfig, cmd string, args []string) error {
	handlers := map[string]func(*globalConfig, []string) error{
		"ls":           cmdLS,
		"stat":         cmdStat,
		"mkdir":        cmdMkdir,
		"rm":           cmdRM,
		"mv":           cmdMV,
		"cp":           cmdCP,
		"upload":       cmdUpload,
		"download":     cmdDownload,
		"search":       cmdSearch,
		"share-create": cmdShareCreate,
		"shares-list":  cmdSharesList,
	}
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		usage()
		return nil
	}
	if h, ok := handlers[cmd]; ok {
		return h(cfg, args)
	}
	return fmt.Errorf("unknown command: %s", cmd)
}

func usage() {
	fmt.Fprintf(os.Stderr, `qnap-cli - Simple CLI for QNAP File Station

Usage:
  qnap-cli [global flags] <command> [command flags]

Global flags:
  -host        QNAP host (or set QNAP_HOST)
  -port        QNAP port (or set QNAP_PORT); 0 = auto
  -user        QNAP username (or set QNAP_USER)
  -pass        QNAP password (or set QNAP_PASS)
  -insecure    Use HTTP / skip TLS verify (or set QNAP_INSECURE=true)
  -timeout     HTTP timeout (default 30s)
  -log-json    Log in JSON (default true)
  -output      Output format: json|text (default json)
  -version     Show version and exit

Commands:
  ls            List files in a directory
  stat          Get info of a file/folder
  mkdir         Create a directory
  rm            Delete file/folder
  mv            Move/rename file/folder
  cp            Copy file to folder
  upload        Upload a local file to remote folder
  download      Download a remote file to local path
  search        Search files by pattern under a path
  share-create  Create a share link for a path
  shares-list   List share links

Examples:
  qnap-cli -host 192.168.1.2 -user admin -pass ****** ls -path /Public
  qnap-cli -host nas.local -user admin -pass ****** upload -local ./a.txt -remote /Public
  qnap-cli -host nas.local -user admin -pass ****** share-create -path /Public/a.txt
`)
}

func main() {
	cfg, rest := parseGlobalFlags(os.Args[1:])
	if cfg.showVersion {
		fmt.Printf("qnap-cli %s (commit %s, built %s)\n", version, commit, date)
		return
	}
	if len(rest) == 0 {
		usage()
		os.Exit(2)
	}

	cmd := rest[0]
	args := rest[1:]

	if err := runCommand(cfg, cmd, args); err != nil {
		log.Fatalf("error: %v", err)
	}
}
