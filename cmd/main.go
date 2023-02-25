package main

import (
	"context"
	"fmt"
	"github.com/broswen/pg-publisher/internal/db"
	"github.com/broswen/pg-publisher/internal/producer"
	"github.com/broswen/pg-publisher/internal/publisher"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// port for api
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = "pg-publisher"
	}

	id := os.Getenv("ID")
	if id == "" {
		id = hostname
	}

	brokers := os.Getenv("BROKERS")
	if brokers == "" {
		log.Fatal().Msgf("kafka brokers are empty")
	}
	publisherTopic := os.Getenv("PUBLISHER_TOPIC")
	if publisherTopic == "" {
		log.Fatal().Msgf("publisher topic is empty")
	}

	dsn := os.Getenv("DSN")
	if dsn == "" {
		log.Fatal().Msgf("postgres DSN is empty")
	}

	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		log.Fatal().Msgf("table name is empty")
	}

	versionColumn := os.Getenv("VERSION_COLUMN")
	if versionColumn == "" {
		log.Fatal().Msgf("version column is empty")
	}

	batchSize := os.Getenv("BATCH_SIZE")
	if batchSize == "" {
		log.Fatal().Msgf("batch size is empty")
	}
	batchSizeVal, err := strconv.ParseInt(batchSize, 10, 64)
	if err != nil {
		log.Fatal().Err(err).Msgf("unable to parse batch size")
	}

	defaultVersion := os.Getenv("DEFAULT_VERSION")
	var defaultVersionVal int64
	if defaultVersion == "" {
		defaultVersionVal = 0
	} else {
		temp, err := strconv.ParseInt(defaultVersion, 10, 64)
		if err != nil {
			log.Fatal().Err(err).Msgf("unable to parse default version")
		}
		defaultVersionVal = temp
	}

	lockname := os.Getenv("LOCK_NAME")
	namespace := os.Getenv("NAMESPACE")

	// port for prometheus
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8081"
	}

	metricsPath := os.Getenv("METRICS_PATH")
	if metricsPath == "" {
		metricsPath = "/metrics"
	}

	eg := errgroup.Group{}

	// start promhttp listener on metrics port
	m := chi.NewRouter()
	m.Handle(metricsPath, promhttp.Handler())
	eg.Go(func() error {
		return http.ListenAndServe(fmt.Sprintf(":%s", metricsPort), m)
	})

	//	initialize database connection
	database, err := db.InitDB(context.Background(), dsn)
	if err != nil {
		log.Fatal().Err(err)
	}

	changesProducer, err := producer.NewRow(hostname, publisherTopic, brokers)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't create changes producer")
	}
	defer changesProducer.Close()

	store, err := publisher.NewPostgresStore(database)
	if err != nil {
		log.Fatal().Err(err).Msg("could not create publisher store")
	}

	changePublisher, err := publisher.NewKafkaPublisher(id, defaultVersionVal, batchSizeVal, tableName, versionColumn, changesProducer, store)

	ctx, cancel := context.WithCancel(context.Background())
	var leading chan struct{}
	if lockname != "" && namespace != "" {
		log.Info().Str("lockname", lockname).Msg("leader election enabled")
		//leader election
		leading = make(chan struct{})
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Panic().Err(err).Msg("couldn't get in cluster config")
		}

		//create new clientset from config
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Panic().Err(err).Msg("couldn't create clientset")
		}

		//define new lock resource
		l := &resourcelock.LeaseLock{
			LeaseMeta: v1.ObjectMeta{
				Name:      lockname,
				Namespace: namespace,
			},
			Client: clientset.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: hostname,
			},
		}

		lec := leaderelection.LeaderElectionConfig{
			Lock:          l,
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod:   2 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					log.Info().Msg("started leading")
					close(leading)
				},
				OnStoppedLeading: func() {
					log.Info().Msg("stopped leading")
					cancel()
				},
				OnNewLeader: func(identity string) {
					log.Info().Str("identity", identity).Msg("new leader")
				},
			},
			WatchDog:        nil,
			ReleaseOnCancel: true,
			Name:            hostname,
		}

		//create new leader elector client
		leaderElector, err := leaderelection.NewLeaderElector(lec)
		if err != nil {
			log.Panic().Err(err).Msg("couldn't create leader elector")
		}

		go func() {
			log.Info().Msg("running leader elector")
			leaderElector.Run(ctx)
		}()
	}

	if leading != nil {
		//if leading is not nil, leader election has been enabled
		//block until leading
		<-leading
	}

	eg.Go(func() error {
		return changePublisher.Run(ctx)
	})

	eg.Go(func() error {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigint
		log.Debug().Str("signal", sig.String()).Msg("received signal")
		cancel()
		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
