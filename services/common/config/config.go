package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config는 각 서비스에서 공통으로 사용하는 환경설정 구조체입니다.
type Config struct {
	Service  ServiceConfig
	HTTP     HTTPConfig
	Log      LogConfig
	Postgres PostgresConfig
	Kafka    KafkaConfig
}

type ServiceConfig struct {
	Name string `envconfig:"SERVICE_NAME" default:"daylog-service"`
}

type HTTPConfig struct {
	Port string `envconfig:"PORT" default:"7000"`
}

type LogConfig struct {
	Level string `envconfig:"LOG_LEVEL" default:"info"`
}

type PostgresConfig struct {
	URI string `envconfig:"POSTGRES_URI"`
}

type KafkaConfig struct {
	Brokers      []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	ActivityTopic string   `envconfig:"KAFKA_TOPIC_ACTIVITY_RAW" default:"activity.raw"`
	GroupID       string   `envconfig:"KAFKA_CONSUMER_GROUP" default:"daylog-consumer"`
}

// MustLoad는 환경변수를 읽어 Config를 반환하며, 실패 시 panic을 발생시킵니다.
func MustLoad(serviceName string) Config {
	cfg, err := Load(serviceName)
	if err != nil {
		panic(err)
	}
	return cfg
}

// Load는 환경변수를 읽어 Config를 반환합니다.
func Load(serviceName string) (Config, error) {
	cfg := Config{}

	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("load env config: %w", err)
	}

	if serviceName != "" {
		cfg.Service.Name = serviceName
	}

	return cfg, nil
}

// Addr는 HTTP 서버가 바인드할 주소를 반환합니다.
func (c Config) Addr() string {
	return ":" + c.HTTP.Port
}

// HasPostgres는 Postgres 연결 정보가 설정되어 있는지 여부를 반환합니다.
func (c Config) HasPostgres() bool {
	return c.Postgres.URI != ""
}

// HasKafka는 Kafka 연결 정보가 설정되어 있는지 여부를 반환합니다.
func (c Config) HasKafka() bool {
	return len(c.Kafka.Brokers) > 0 && c.Kafka.Brokers[0] != ""
}
