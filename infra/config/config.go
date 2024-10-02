package config

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"regexp"
	"strings"

	tpl "github.com/cloudcopper/misc/env/template"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/ports"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repos map[string]models.Repo
}

func (c *Config) String() string {
	s := ""
	for k, repo := range c.Repos {
		s += fmt.Sprintf("%v:\n", k)
		if k == repo.ID {
			s += fmt.Sprintf("    #ID: %v\n", repo.ID)
		} else {
			s += fmt.Sprintf("    ID: %v\n", repo.ID)
		}
		s += fmt.Sprintf("    name: %v\n", repo.Name)
		s += fmt.Sprintf("    description: %v\n", repo.Description)
		s += fmt.Sprintf("    input: %v\n", repo.Input)
		s += fmt.Sprintf("    storage: %v\n", repo.Storage)
		s += fmt.Sprintf("    retention: %v\n", repo.Retention)
		s += fmt.Sprintf("    broken: %v\n", repo.Broken)
	}
	return strings.TrimSuffix(s, "\n")
}

const refRepoID = "${REPO_ID}"

var (
	ReposConfigFileName   = "swamp_repos.yml"
	TopRootFileSystemPath = ""
)

func LoadConfig(log ports.Logger, fs fs.ReadFileFS) (*Config, error) {
	config, err := loadReposConfig(log, fs, ReposConfigFileName)
	if err != nil {
		return config, err
	}
	config = processReposConfigs(log, config)

	// dump effective config
	dump := strings.Split(config.String(), "\n")
	for _, s := range dump {
		log.Debug(s)
	}
	return config, nil
}

// The loadReposConfig reads named repos configs file from given fs,
// execute file as env template,
// and unmarshal result to the config
func loadReposConfig(log ports.Logger, fs fs.ReadFileFS, fileName string) (*Config, error) {
	log.Info("loading repos config", slog.String("fileName", fileName))
	blob, err := os.ReadFile(fileName)
	if err != nil {
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
	cfg := &Config{}
	err = yaml.Unmarshal([]byte(s), &cfg.Repos)
	return cfg, err
}

// The processReposConfigs returns only meaningful repo configuration
// with correct @refRepoID macro
func processReposConfigs(log ports.Logger, config *Config) *Config {
	ret := &Config{
		Repos: make(map[string]models.Repo),
	}
	reValidID := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-\_\.]{2,}$`)

	for k, v := range config.Repos {
		log := log.With(slog.String("configID", k))

		// Skip IDs starting with _
		// Sort of special meaning
		if strings.HasPrefix(k, "_") {
			continue
		}

		// Correct ID
		if v.ID == "" {
			v.ID = refRepoID
		}
		v.ID = strings.ReplaceAll(string(v.ID), refRepoID, k)
		log = log.With(slog.Any("repoID", v.ID))
		if !reValidID.MatchString(string(v.ID)) {
			log.Error("skip - invalid repo id")
			continue
		}

		// Replace all entry of @refRepoID to ID
		replaceRefRepoID := func(s string) string {
			return strings.ReplaceAll(s, refRepoID, string(v.ID))
		}
		v.Name = replaceRefRepoID(v.Name)
		v.Description = replaceRefRepoID(v.Description)
		v.Input = replaceRefRepoID(v.Input)
		v.Storage = replaceRefRepoID(v.Storage)
		v.Broken = replaceRefRepoID(v.Broken)

		if v.Storage == "" {
			log.Warn("skip - repo has no storage location")
			continue
		}

		if v.Input == "" {
			log.Warn("repo has no input - read-only repo")
		}

		ret.Repos[k] = v
	}

	// TODO Check multiple repos has same input
	// TODO Check multiple repos has same storage

	return ret
}
