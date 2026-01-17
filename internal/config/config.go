package config

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
)

type Config struct {
    ListenAddr         string   `json:"listenAddr"`
    AllowedTokens      []string `json:"allowedTokens"`
    LogDir             string   `json:"logDir"`
    PairTimeoutSeconds int      `json:"pairTimeoutSeconds"`
    MaxMessageLogBytes int      `json:"maxMessageLogBytes"`

    // Optional upstream auto-connect settings
    AutoConnect  bool   `json:"autoConnect"`
    UpstreamURL  string `json:"upstreamUrl"`
    UpstreamRole string `json:"upstreamRole"`
    UpstreamToken string `json:"upstreamToken"`

    // Optional console test mode: connects locally as newcli and bridges stdin/stdout
    ConsoleTest bool `json:"consoleTest"`
}

func Default() Config {
    return Config{
        ListenAddr:         ":8080",
        AllowedTokens:      nil,
        LogDir:             "",
        PairTimeoutSeconds: 0,
        MaxMessageLogBytes: 2048,
        AutoConnect:        false,
        UpstreamURL:        "",
        UpstreamRole:       "codex",
        UpstreamToken:      "",
        ConsoleTest:        false,
    }
}

// DefaultPath returns the default config path under %USERPROFILE%\.codex\relay.json on Windows,
// or $HOME/.codex/relay.json otherwise.
func DefaultPath() string {
    // Prefer USERPROFILE on Windows
    if up := os.Getenv("USERPROFILE"); up != "" {
        return filepath.Join(up, ".codex", "relay.json")
    }
    if home, err := os.UserHomeDir(); err == nil && home != "" {
        return filepath.Join(home, ".codex", "relay.json")
    }
    return "relay.json"
}

func Load(path string) (Config, error) {
    cfg := Default()

    if path == "" {
        path = DefaultPath()
    }

    // If a directory path is provided, look for relay.json inside it.
    if fi, err := os.Stat(path); err == nil && fi.IsDir() {
        path = filepath.Join(path, "relay.json")
    }

    f, err := os.Open(path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return cfg, nil
        }
        return cfg, fmt.Errorf("open config: %w", err)
    }
    defer f.Close()

    dec := json.NewDecoder(f)
    if err := dec.Decode(&cfg); err != nil {
        return cfg, fmt.Errorf("decode config JSON: %w", err)
    }
    return cfg, nil
}
