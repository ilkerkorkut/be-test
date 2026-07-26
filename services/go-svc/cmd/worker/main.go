package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flotaro/be-test/services/go-svc/config"
	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"github.com/flotaro/be-test/services/go-svc/internal/userdata"
	"github.com/flotaro/be-test/services/go-svc/internal/worker"
	"github.com/flotaro/be-test/services/go-svc/pkg/kafka"
	"github.com/flotaro/be-test/services/go-svc/pkg/logging"
	"go.uber.org/zap"
)

func main() {
	configuration := config.NewConfigurationManager()
	logger, err := logging.InitLogger(configuration.GetAppConfig().LogLevel)
	if err != nil {
		log.Println("failed to initialize logger")
		return
	}

	initialWallets, err := userdata.LoadInitialWallets(configuration.GetDataConfig().FolderPath)
	if err != nil {
		logger.Fatal("failed to load initial wallet data", zap.Error(err))
	}
	db, err := database.NewJSONStore(configuration.GetDatabaseConfig().Path, initialWallets)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	producer, err := kafka.NewProducer(configuration.GetKafkaConfig().Brokers)
	if err != nil {
		logger.Fatal("failed to initialize kafka producer", zap.Error(err))
	}
	defer producer.Close()

	placementConsumer, err := worker.NewPlacementConsumer(configuration.GetKafkaConfig().Brokers, configuration.GetKafkaConfig().ConsumerGroup+"-placement", db, logger)
	if err != nil {
		logger.Fatal("failed to initialize placement consumer", zap.Error(err))
	}
	defer placementConsumer.Close()
	settlementConsumer, err := worker.NewSettlementConsumer(configuration.GetKafkaConfig().Brokers, configuration.GetKafkaConfig().ConsumerGroup+"-settlement", worker.NewSettlementService(db, logger), logger)
	if err != nil {
		logger.Fatal("failed to initialize settlement consumer", zap.Error(err))
	}
	defer settlementConsumer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go service.NewOutboxDispatcher(db, producer, logger).Run(ctx, time.Second)
	go worker.NewSettlementScheduler(db, logger).Run(ctx, time.Second)
	go func() {
		if runErr := placementConsumer.Run(ctx); runErr != nil {
			logger.Error("placement consumer stopped with an error", zap.Error(runErr))
		}
	}()

	if err := settlementConsumer.Run(ctx); err != nil {
		logger.Error("settlement consumer stopped with an error", zap.Error(err))
	}
}
