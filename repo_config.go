package main

import (
	"embed"
	"log/slog"
	"os"
	"strings"
	"time"

	tpl "github.com/cloudcopper/misc/env/template"

	"gopkg.in/yaml.v3"
)

type RepoConfigs map[string]RepoConfig

type RepoConfig struct {
	Defaults  string
	Name      string
	Input     string
	Meta      string
	Seal      string
	Storage   string
	Retention *time.Duration
	Broken    string
}

const defaultsNodeName = "defaults"
const specialRepoName = "${REPO_NAME}"
const defaultRepoConfigsFileName = "swamp_repos.yml"

var repoConfigsFileName = getEnvDefault("SWAMP_REPO_CONFIG", defaultRepoConfigsFileName)

//go:embed *.yml
var fs embed.FS

// LoadRepoConfigs reads repo configs named file from filesystem,
// optionally fallback to embedded filesystem,
// execute file as env template,
// and unmarshal result to the config
func LoadRepoConfigs(log *Logger, fileName string) (RepoConfigs, error) {
	log.Info("loading repo config", slog.String("fileName", fileName))
	// try load from real filesystem
	blob, err := os.ReadFile(fileName)
	if err != nil {
		// fallback to read from embedded filesystem
		log.Info("fallback to embedded repo config", slog.String("fileName", fileName))
		blob, err = fs.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
	}

	// parse config as template
	t, err := tpl.Parse(string(blob))
	if err != nil {
		return nil, err
	}
	// execute template
	s, err := t.Execute()
	if err != nil {
		return nil, err
	}

	// unmrashal config
	cfg := RepoConfigs{}
	err = yaml.Unmarshal([]byte(s), &cfg)
	return cfg, err
}

func LoadRepoConfigsDefaults(log *Logger, config RepoConfigs) RepoConfigs {
	ret := RepoConfigs{}

	for k, v := range config {
		log := log.With(slog.String("config", k))
		v.Name = strings.TrimSpace(v.Name)
		if v.Name == "" {
			log.Warn("skip config - has no name")
			continue
		}

		// find defaults
		nodeName := defaultsNodeName
		if v.Defaults != "" {
			nodeName = v.Defaults
		}
		def, ok := config[nodeName]
		if !ok {
			log.Error("skip config - no defaults found!!!", slog.String("defaults", nodeName))
			continue
		}

		if v.Input == "" {
			v.Input = def.Input
		}
		if v.Name == specialRepoName && v.Input != "" && !strings.Contains(v.Input, specialRepoName) {
			log.Error("skip config - special name input missmatch!!!", slog.String("name", v.Name), slog.String("input", v.Input))
			continue
		}
		if v.Input != "" {
			v.Input = strings.ReplaceAll(v.Input, specialRepoName, v.Name)
		}
		if v.Input == "" {
			log.Warn("config repo has no input - read-only repo", slog.String("name", v.Name))
		}

		if v.Meta == "" {
			v.Meta = def.Meta
		}
		if v.Seal == "" {
			v.Seal = def.Seal
		}

		if v.Storage == "" {
			v.Storage = def.Storage
		}
		if v.Storage == "" {
			log.Warn("skip config - has no storage location")
		}
		v.Storage = strings.ReplaceAll(v.Storage, specialRepoName, v.Name)

		if v.Retention == nil {
			v.Retention = def.Retention
		}
		if v.Broken == "" {
			v.Broken = def.Broken
		}

		ret[k] = v
	}

	// TODO Check multiple repos has same storage

	return ret
}
