package producer

import (
	"github.com/rs/zerolog/log"
)

type Row interface {
	Submit(row map[string]interface{}) error
	Close() error
}

func NewRow(id, topic, brokers string) (Row, error) {
	if brokers == "" || brokers == "logger" {
		log.Warn().Str("id", id).Str("topic", topic).Msg("BROKERS is empty, using log producer")
		return NewLogProducer(id, topic)
	} else {
		return NewKafkaProducer(id, topic, brokers)
	}
}
