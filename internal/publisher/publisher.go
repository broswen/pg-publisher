package publisher

import (
	"context"
	"errors"
	"github.com/broswen/pg-publisher/internal/producer"
	"github.com/rs/zerolog/log"
	"time"
)

type Publisher interface {
	Run(ctx context.Context) error
}

type KafkaPublisher struct {
	id                   string
	lastPublishedVersion int64
	store                Store
	batchSize            int64
	producer             producer.Row
	tableName            string
	versionColumn        string
}

func NewKafkaPublisher(id string, defaultVersion, batchSize int64, tableName, versionColumn string, producer producer.Row, store Store) (*KafkaPublisher, error) {
	if id == "" {
		return nil, errors.New("publisher id is empty")
	}
	return &KafkaPublisher{
		id:                   id,
		store:                store,
		lastPublishedVersion: defaultVersion,
		producer:             producer,
		tableName:            tableName,
		versionColumn:        versionColumn,
		batchSize:            batchSize,
	}, nil
}

func (p *KafkaPublisher) Run(ctx context.Context) error {
	log.Info().Str("id", p.id).Msg("starting publisher")
	lastPublishedVersion, err := p.store.GetLastPublishedVersion(ctx, p.id)
	if err != nil {
		log.Error().Err(err).Str("id", p.id).Msg("error getting last published version")
		if !errors.As(err, &ErrNotFound{}) {
			return errors.New("unable to get last published version")
		}
		//if version not found in DB, use defaultVersion
		log.Warn().Int64("lastPublishedVersion", p.lastPublishedVersion).Msg("could not get last published version, using default")
		lastPublishedVersion = p.lastPublishedVersion
	} else {
		log.Info().Int64("lastPublishedVersion", lastPublishedVersion).Msg("got last published version")
	}
	p.lastPublishedVersion = lastPublishedVersion

	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			start := time.Now()
			log.Info().Str("id", p.id).Msg("checking latest version")
			latestVersion, err := p.store.GetLatestVersion(ctx, p.tableName, p.versionColumn)
			if err != nil {
				log.Error().Err(err).Str("id", p.id).Str("tableName", p.tableName).Str("versionColumn", p.versionColumn).Msg("error getting latest version")
				continue
			}
			if p.lastPublishedVersion >= latestVersion {
				log.Info().Str("id", p.id).Int64("version", p.lastPublishedVersion).Msg("no changes")
				continue
			}
			log.Info().Str("id", p.id).Msg("change detected")

			more := true
			for more {
				rows, err := p.store.ListFromVersion(ctx, p.tableName, p.versionColumn, p.lastPublishedVersion, p.batchSize)
				if err != nil {
					log.Error().Err(err).Str("id", p.id).Msg("error listing changes")
					continue
				}
				//continue loop if batch size wasn't full (potentially more in table)
				more = int64(len(rows)) == p.batchSize
				for _, row := range rows {
					err = p.producer.Submit(row)
					if err != nil {
						log.Error().Err(err).Str("id", p.id).Msg("error submitting row")
						PublishErrors.Inc()
						break
					}
					version, ok := row[p.versionColumn]
					if !ok {
						log.Error().Err(err).Str("id", p.id).Str("versionColumn", p.versionColumn).Msg("version column not found on row")
						PublishErrors.Inc()
						break
					}

					if val, ok := version.(int64); ok {
						p.lastPublishedVersion = val
					} else {
						log.Error().Err(err).Str("id", p.id).Interface("version", val).Msg("version column value not int64")
						PublishErrors.Inc()
						break
					}
					ProvisionedVersions.Inc()
				}

				LastPublishedVersion.Set(float64(p.lastPublishedVersion))
				err = p.store.SetLastPublishedVersion(ctx, p.id, p.lastPublishedVersion)
				if err != nil {
					log.Error().Err(err).Str("id", p.id).Msg("error setting last published version")
					continue
				}
			}
			TickLatency.Observe(float64(time.Since(start).Milliseconds()))
			Ticks.Inc()

		case <-ctx.Done():
			log.Info().Str("id", p.id).Msg("stopping publisher")
			log.Info().Str("id", p.id).Int64("version", p.lastPublishedVersion).Msg("setting last published version")
			newCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			err := p.store.SetLastPublishedVersion(newCtx, p.id, p.lastPublishedVersion)
			if err != nil {
				log.Error().Err(err).Str("id", p.id).Int64("version", p.lastPublishedVersion).Msg("error setting last published version")
			}
			return nil
		}
	}
}
