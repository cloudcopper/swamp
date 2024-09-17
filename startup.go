package swamp

import (
	"log/slog"

	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/ports"
)

func startup(log ports.Logger, config *infra.Config, bus ports.EventBus, repoRepository domain.RepoRepository) error {
	//
	// Create repo models
	//
	for k, repo := range config.Repos {
		log := log.With(slog.String("config", k), slog.Any("repoID", repo.ID))

		// Create repo model in repository
		if err := repoRepository.Create(&repo); err != nil {
			log.Error("unable create repo record", slog.Any("err", err))
			return lib.NewErrorCode(err, retCreateRepoRecordError)
		}
		// Emit event on repo model updated and input updated
		bus.Pub(ports.TopicRepoUpdated, ports.Event{repo.ID})
		bus.Pub(ports.TopicInputUpdated, ports.Event{repo.Input})
	}

	return nil
}
