package config

import (
	"log/slog"
	"testing"

	"github.com/cloudcopper/swamp/domain/models"
	"github.com/stretchr/testify/require"
)

func TestRemoveSameRepos(t *testing.T) {
	testCases := []struct {
		desc string
		in   map[string]*models.Repo
		out  map[string]*models.Repo
	}{
		{
			desc: "well configured",
			in: map[string]*models.Repo{
				"repo1": {
					Input:   "/input/repo1",
					Storage: "/storage/repo1",
				},
				"repo2": {
					Input:   "/input/repo2",
					Storage: "/storage/repo2",
				},
				"repo3": {
					Input:   "/input/repo3",
					Storage: "/storage/repo3",
				},
			},
			out: map[string]*models.Repo{
				"repo1": {
					Input:   "/input/repo1",
					Storage: "/storage/repo1",
				},
				"repo2": {
					Input:   "/input/repo2",
					Storage: "/storage/repo2",
				},
				"repo3": {
					Input:   "/input/repo3",
					Storage: "/storage/repo3",
				},
			},
		},
		{
			desc: "two input dups",
			in: map[string]*models.Repo{
				"repo1": {
					Input:   "/input/repo",
					Storage: "/storage/repo1",
				},
				"repo2": {
					Input:   "/input/repo",
					Storage: "/storage/repo2",
				},
			},
			out: map[string]*models.Repo{},
		},
		{
			desc: "two storage dups",
			in: map[string]*models.Repo{
				"repo1": {
					Input:   "/input/repo1",
					Storage: "/storage/repo",
				},
				"repo2": {
					Input:   "/input/repo2",
					Storage: "/storage/repo",
				},
			},
			out: map[string]*models.Repo{},
		},
		{
			desc: "overlapped dups",
			in: map[string]*models.Repo{
				"repo1": {
					Input:   "/input/repo",
					Storage: "/storage/repo1",
				},
				"repo2": {
					Input:   "/input/repo",
					Storage: "/storage/repo",
				},
				"repo3": {
					Input:   "/input/repo3",
					Storage: "/storage/repo",
				},
			},
			out: map[string]*models.Repo{},
		},
		{
			desc: "only one normal",
			in: map[string]*models.Repo{
				"repo1": {
					Input:   "/input/repo",
					Storage: "/storage/repo",
				},
				"repo2": {
					Input:   "/input/repo",
					Storage: "/storage/repo",
				},
				"repo3": {
					Input:   "/input/repo3",
					Storage: "/storage/repo3",
				},
			},
			out: map[string]*models.Repo{
				"repo3": {
					Input:   "/input/repo3",
					Storage: "/storage/repo3",
				},
			},
		},
		{
			desc: "two nested",
			in: map[string]*models.Repo{
				"repo1": {
					Input:   "/input/repo/repo1",
					Storage: "/storage/repo1",
				},
				"repo2": {
					Input:   "/input/repo",
					Storage: "/storage/repo2",
				},
				"repo3": {
					Input:   "/input/repo3",
					Storage: "/storage/repo3",
				},
			},
			out: map[string]*models.Repo{
				"repo3": {
					Input:   "/input/repo3",
					Storage: "/storage/repo3",
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert := require.New(t)
			out := removeSameRepos(slog.Default(), tC.in)
			assert.Equal(tC.out, out)
		})
	}
}
