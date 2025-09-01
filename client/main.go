package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/common"
)

var log = logging.MustGetLogger("log")

// InitConfig: carga config de archivo + env (prefijo CLI_)
func InitConfig() (*viper.Viper, error) {
	v := viper.New()

	v.SetDefault("dataset.path", "")

	// Env con prefijo CLI_
	v.AutomaticEnv()
	v.SetEnvPrefix("cli")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = v.BindEnv("id")
	_ = v.BindEnv("server", "address")
	_ = v.BindEnv("loop", "period")
	_ = v.BindEnv("loop", "amount")
	_ = v.BindEnv("log", "level")

	// Archivo de config
	v.SetConfigFile("./config.yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Configuration could not be read from config file. Using env variables instead")
	}

	// Validación de loop.period (aunque no lo usemos en Ej.6, mantiene compat)
	if _, err := time.ParseDuration(v.GetString("loop.period")); err != nil {
		return nil, errors.Wrapf(err, "Could not parse CLI_LOOP_PERIOD env var as time.Duration.")
	}

	return v, nil
}

// InitLogger
func InitLogger(logLevel string) error {
	baseBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(`%{time:2006-01-02 15:04:05} %{level:.5s}     %{message}`)
	backendFormatter := logging.NewBackendFormatter(baseBackend, format)

	backendLeveled := logging.AddModuleLevel(backendFormatter)
	logLevelCode, err := logging.LogLevel(logLevel)
	if err != nil {
		return err
	}
	backendLeveled.SetLevel(logLevelCode, "")
	logging.SetBackend(backendLeveled)
	return nil
}

// PrintConfig (podés ampliar si querés ver batch/dataset)
func PrintConfig(v *viper.Viper) {
	log.Infof("action: config | result: success | client_id: %s | server_address: %s | loop_amount: %v | loop_period: %v | log_level: %s",
		v.GetString("id"),
		v.GetString("server.address"),
		v.GetInt("loop.amount"),
		v.GetDuration("loop.period"),
		v.GetString("log.level"),
	)
}

func signalHandler(client *common.Client) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Infof("action: signal_received | result: success")
		client.Close()
		os.Exit(0)
	}()
}

func main() {
	v, err := InitConfig()
	if err != nil {
		log.Criticalf("%s", err)
	}

	if err := InitLogger(v.GetString("log.level")); err != nil {
		log.Criticalf("%s", err)
	}

	PrintConfig(v)
	
	clientConfig := common.ClientConfig{
		ServerAddress:  v.GetString("server.address"),
		ID:             v.GetString("id"),
		LoopAmount:     v.GetInt("loop.amount"),
		LoopPeriod:     v.GetDuration("loop.period"),
		DatasetPath:    "/agency.csv",
		BatchMaxAmount: v.GetInt("batch.maxAmount"),
	}

	client := common.NewClient(clientConfig)
	signalHandler(client)

	if err := client.SendBatchesFromDataset(); err != nil {
		log.Criticalf("action: apuestas_enviadas | result: fail | error: %v", err)
	}
}