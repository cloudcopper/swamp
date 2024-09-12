package swamp

import (
	"io"
	"log/slog"
	"testing"

	testifyAssert "github.com/stretchr/testify/assert"
)

func TestFileConfigYml(t *testing.T) {
	assert := testifyAssert.New(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil)) // https://www.youtube.com/watch?v=i1bDIyIaxbE

	// The repo shall has file swamp_repos.yml
	cfg, err := LoadRepoConfigs(log, repoConfigsFileName)
	assert.NoError(err)

	// The swamp_repos.yml shall define at least default
	assert.Contains(cfg, defaultsNodeName)
}
