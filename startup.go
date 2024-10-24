package swamp

import (
	"log/slog"

	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/infra/config"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

func startup(log ports.Logger, cfg *config.Config, bus ports.EventBus, repoRepository domain.RepoRepository) error {
	//
	// Create repo models
	//
	for k, repo := range cfg.Repos {
		log := log.With(slog.String("config", k), slog.Any("repoID", repo.RepoID))

		// Create repo model in repository
		if err := repoRepository.Create(repo); err != nil {
			log.Error("unable create repo record", slog.Any("err", err))
			return lib.NewErrorCode(err, errors.RetCreateRepoRecordError)
		}
		// Emit event on repo model updated and input updated
		bus.Pub(ports.TopicRepoUpdated, ports.Event{repo.RepoID})
		bus.Pub(ports.TopicInputUpdated, ports.Event{repo.Input})
	}

	return nil
}
