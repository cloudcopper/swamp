package config

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"time"

	tpl "github.com/cloudcopper/misc/env/template"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repos map[string]*models.Repo
}

func (c *Config) String() string {
	s := ""
	for k, repo := range c.Repos {
		s += fmt.Sprintf("%v:\n", k)
		if k == repo.RepoID {
			s += fmt.Sprintf("    #RepoID: %v\n", repo.RepoID)
		} else {
			s += fmt.Sprintf("    RepoID: %v\n", repo.RepoID)
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
	Listen                = ":8080"
	ReposConfigFileName   = "swamp_repos.yml"
	TopRootFileSystemPath = ""
	TimerExpiredStart     = 30 * time.Minute
	TimerExpiredInterval  = 1 * time.Minute
	TimerExpiredLimit     = 1
	TimerBrokenStart      = 30 * time.Minute
	TimerBrokenInterval   = 1 * time.Minute
	TimerBrokenLimit      = 1
)

func LoadConfig(log ports.Logger, f fs.ReadFileFS) (*Config, error) {
	config, err := loadReposConfig(log, f, ReposConfigFileName)
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
func loadReposConfig(log ports.Logger, f fs.ReadFileFS, fileName string) (*Config, error) {
	log.Info("loading repos config", slog.String("fileName", fileName))
	blob, err := os.ReadFile(fileName)
	if err != nil {
		blob, err = f.ReadFile(fileName)
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
func processReposConfigs(log ports.Logger, cfg *Config) *Config {
	ret := &Config{
		Repos: make(map[string]*models.Repo),
	}

	for k, v := range cfg.Repos {
		log := log.With(slog.String("configID", k))

		// Skip IDs starting with _
		// Sort of special meaning
		if strings.HasPrefix(k, "_") {
			continue
		}

		// Correct ID
		if v.RepoID == "" {
			v.RepoID = refRepoID
		}
		v.RepoID = strings.ReplaceAll(string(v.RepoID), refRepoID, k)
		log = log.With(slog.Any("repoID", v.RepoID))
		if !lib.IsValidID(string(v.RepoID)) {
			log.Error("skip - invalid repo id")
			continue
		}

		// Replace all entry of @refRepoID to ID
		replaceRefRepoID := func(s string) string {
			return strings.ReplaceAll(s, refRepoID, string(v.RepoID))
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

	// Check multiple repos has same input/storage
	ret.Repos = removeSameRepos(log, ret.Repos)

	return ret
}

func removeSameRepos(log ports.Logger, in map[string]*models.Repo) map[string]*models.Repo {
	out := map[string]*models.Repo{}

	isNested := func(a, b string) bool {
		if !strings.HasSuffix(a, "/") {
			a += "/"
		}
		if !strings.HasSuffix(b, "/") {
			b += "/"
		}
		return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
	}

	for k1, v1 := range in {
		isDup := false

		for k2, v2 := range in {
			if k1 == k2 {
				continue
			}
			switch {
			case v1.Input == v2.Input:
				isDup = true
				log.Error("duplicated config detected", slog.Any("repoID", k1), slog.Any("input", v1.Input))
			case v1.Storage == v2.Storage:
				isDup = true
				log.Error("duplicated config detected", slog.Any("repoID", k1), slog.Any("storage", v1.Storage))
			case isNested(v1.Storage, v2.Storage):
				isDup = true
				log.Error("nested config detected", slog.Any("repoID", k1), slog.Any("storage", v1.Storage))
			case isNested(v1.Input, v2.Input):
				isDup = true
				log.Error("nested config detected", slog.Any("repoID", k1), slog.Any("storage", v1.Storage))
			}
			if isDup {
				break
			}
		}

		if isDup {
			continue
		}
		out[k1] = v1
	}

	return out
}
