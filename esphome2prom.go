package main

import (
	"context"
	"embed"
	"github.com/XANi/esphome2prom/config"
	"github.com/XANi/esphome2prom/queue"
	"github.com/XANi/esphome2prom/web"
	"github.com/XANi/go-yamlcfg"
	"github.com/XANi/goneric"
	"github.com/efigence/go-mon"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
	"os"
)

var version string
var log *zap.SugaredLogger
var debug = true
var exit = make(chan error, 1)

// /* embeds with all files, just dir/ ignores files starting with _ or .
//
//go:embed static templates
var embeddedWebContent embed.FS

func init() {
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	// naive systemd detection. Drop timestamp if running under it
	if os.Getenv("JOURNAL_STREAM") != "" {
		consoleEncoderConfig.TimeKey = ""
	}
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return (lvl < zapcore.ErrorLevel) != (lvl == zapcore.DebugLevel && !debug)
	})
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, os.Stderr, lowPriority),
		zapcore.NewCore(consoleEncoder, os.Stderr, highPriority),
	)
	logger := zap.New(core)
	if debug {
		logger = logger.WithOptions(
			zap.Development(),
			zap.AddCaller(),
			zap.AddStacktrace(highPriority),
		)
	} else {
		logger = logger.WithOptions(
			zap.AddCaller(),
		)
	}
	log = logger.Sugar()

}

func main() {
	defer log.Sync()
	// register internal stats
	mon.RegisterGcStats()
	app := &cli.Command{
		Name:        "foobar",
		Aliases:     nil,
		Usage:       "",
		UsageText:   "",
		ArgsUsage:   "",
		Version:     "",
		Description: "do foo to bar",
		Flags:       nil,
		Commands:    nil,
		HideHelp:    true,
	}
	app.Name = "esphome2prom"
	app.Description = "Convert esphome metric to prometheus write protocol"
	app.Version = version
	app.HideHelp = true
	log.Infof("Starting %s version: %s", app.Name, version)
	app.Flags = []cli.Flag{
		&cli.BoolFlag{Name: "help, h", Usage: "show help"},
		&cli.BoolFlag{Name: "debug, d", Usage: "enable debug logs"},
		&cli.StringFlag{Name: "config, c",
			Usage: "config file. Will be created if it does not exist",
		},
		&cli.StringFlag{
			Name:  "listen-addr",
			Usage: "Listen addr",
			Sources: cli.NewValueSourceChain(
				cli.EnvVar("LISTEN_ADDR"),
			),
		},
		&cli.StringFlag{
			Name:     "mqtt-addr",
			Usage:    "mqtt broker address",
			Required: true,
			Sources: cli.NewValueSourceChain(
				cli.EnvVar("MQTT_ADDR"),
			),
		},
		&cli.StringFlag{
			Name:  "prometheus-write-url",
			Usage: "prometheus write protocol url",
			Sources: cli.NewValueSourceChain(
				cli.EnvVar("PROMETHEUS_WRITE_URL"),
			),
		},
		&cli.StringFlag{
			Name:  "pprof-addr",
			Value: "",
			Usage: "address to run pprof on, disabled by default",
		},
		&cli.StringFlag{
			Name:  "prefix",
			Value: "",
			Usage: "prefix for metrics name",
		},
		&cli.StringMapFlag{
			Name: "extra-labels",
			Value: map[string]string{
				"host": goneric.Must(os.Hostname()),
			},
			Usage: "comma separated key=value pairs of additional prometheus labels",
		},
	}
	app.Action = func(ctx context.Context, c *cli.Command) error {
		if c.Bool("help") {
			cli.ShowAppHelp(c)
			os.Exit(1)
		}
		if c.String("prometheus-write-url") == "" && c.String("listen-addr") == "" {
			log.Panic("must specify --prometheus-write-url or --listen-addr")
		}

		cfgFiles := []string{
			c.String("config"),
		}
		cfg := config.Config{
			MQTTAddress:        c.String("mqtt-addr"),
			PrometheusWriteURL: c.String("prometheus-write-url"),
			ListenAddress:      c.String("listen-addr"),
			Debug:              c.Bool("debug"),
			PProfAddress:       c.String("pprof-addr"),
			ExtraLabels:        c.StringMap("extra-labels"),
			PrometheusPrefix:   c.String("prometheus-prefix"),
		}
		if c.String("config") != "" {
			err := yamlcfg.LoadConfig(cfgFiles, &cfg)
			if err != nil {
				log.Fatal(err)
			}
		}
		debug = cfg.Debug
		log.Debug("debug enabled")

		var webDir fs.FS
		webDir = embeddedWebContent
		if st, err := os.Stat("./static"); err == nil && st.IsDir() {
			if st, err := os.Stat("./templates"); err == nil && st.IsDir() {
				webDir = os.DirFS(".")
				log.Infof(`detected directories "static" and "templates", using local static files instead of ones embedded in binary`)
			}
		}

		os.DirFS(".")
		if len(cfg.ListenAddress) > 0 {
			w, err := web.New(web.Config{
				Logger:     log,
				ListenAddr: cfg.ListenAddress,
			}, webDir)
			_ = w
			if err != nil {
				log.Panicf("error starting web listener: %s", err)
			}
		}
		if len(cfg.PProfAddress) > 0 {
			log.Infof("listening pprof on %s", cfg.PProfAddress)
			go func() {
				log.Errorf("failed to start debug listener: %s (ignoring)", http.ListenAndServe(cfg.PProfAddress, nil))
			}()
		}
		_, err := queue.New(&queue.Config{
			MQTTAddr:    cfg.MQTTAddress,
			Logger:      log.Named("mq"),
			ExtraLabels: cfg.ExtraLabels,
			Prefix:      cfg.PrometheusPrefix,
			Debug:       debug,
		})
		if err != nil {
			log.Panicf("error starting queue listener: %s", err)
		}
		return <-exit
	}
	// to sort do that
	// sort.Sort(cli.FlagsByName(app.Flags))
	// sort.Sort(cli.CommandsByName(app.Commands))
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
