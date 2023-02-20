package producer

import (
	"github.com/rs/zerolog/log"
)

type LogProducer struct {
	id    string
	topic string
}

func NewLogProducer(id, topic string) (*LogProducer, error) {
	return &LogProducer{
		id:    id,
		topic: topic,
	}, nil
}

func (p *LogProducer) Close() error {
	return nil
}

func (p *LogProducer) Submit(row map[string]interface{}) error {
	log.Debug().Str("producer_id", p.id).Str("topic", p.topic).Msgf("%#v", row)
	return nil
}

type HealthcheckLogProducer struct {
	id    string
	topic string
}
