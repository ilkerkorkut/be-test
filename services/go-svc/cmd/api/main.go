package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flotaro/be-test/services/go-svc/config"
	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"github.com/flotaro/be-test/services/go-svc/internal/userdata"
	"github.com/flotaro/be-test/services/go-svc/pkg/kafka"
	"github.com/flotaro/be-test/services/go-svc/pkg/logging"
	"go.uber.org/zap"
)

func main() {
	cm := config.NewConfigurationManager()

	logger, err := logging.InitLogger(
		cm.GetAppConfig().LogLevel,
	)
	if err != nil {
		log.Println("failed to initialize logger")
		return
	}

	logger.With(
		zap.String("app_name", cm.GetAppConfig().Name),
		zap.String("env", cm.GetAppConfig().Env),
		zap.String("log_level", cm.GetAppConfig().LogLevel),
	).Info("starting application")

	initialWallets, err := userdata.LoadInitialWallets(cm.GetDataConfig().FolderPath)
	if err != nil {
		logger.Fatal("failed to load initial wallet data", zap.Error(err))
	}
	db, err := database.NewJSONStore(cm.GetDatabaseConfig().Path, initialWallets)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()
	producer, err := kafka.NewProducer(cm.GetKafkaConfig().Brokers)
	if err != nil {
		logger.Fatal("failed to initialize kafka producer", zap.Error(err))
	}
	defer producer.Close()
	dispatchContext, cancelDispatch := context.WithCancel(context.Background())
	defer cancelDispatch()
	go service.NewOutboxDispatcher(db, producer, logger).Run(dispatchContext, time.Second)

	app, err := initApplication(&application{
		Logger:   logger,
		Config:   cm,
		Database: db,
	})
	if err != nil {
		logger.Fatal("failed to initialize application", zap.Error(err))
	}

	go func() {
		if serveErr := app.Listen(fmt.Sprintf(":%s", cm.GetAppConfig().Port)); serveErr != nil {
			logger.Fatal("failed to start server")
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	<-c

	if shutdownErr := app.Shutdown(); shutdownErr != nil {
		logger.Fatal("failed to shutdown server")
	}

	logger.Info("application shutdown gracefully")

}
