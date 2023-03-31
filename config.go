package main

import (
	"errors"
	"flag"
	"os"

	logrus "github.com/sirupsen/logrus"
)

type Config struct {
	PubSubTopic   string
	ContentTopic  string
	MsgPerSecond  uint64
	MsgSizeKb     uint64
	BootstrapNode string
}

// By default the release is a custom build. CI takes care of upgrading it with
// go build -v -ldflags="-X 'github.com/xxx/yyy/config.ReleaseVersion=x.y.z'"
var ReleaseVersion = "custom-build"

func NewCliConfig() (*Config, error) {
	var version = flag.String("version", "", "Prints the release version and exits")
	var pubSubTopic = flag.String("pubsub-topic", "", "PubSub topic to publish messages to")
	var contentTopic = flag.String("content-topic", "", "Content topic to publish messages to")
	var msgPerSecond = flag.Uint64("msg-per-second", 0, "Number of messages to publish per second")
	var msgSizeKb = flag.Uint64("msg-size-kb", 0, "Size of messages to publish in kilobytes")
	var bootstrapNode = flag.String("bootstrap-node", "", "Bootstrap node to connect to")

	flag.Parse()

	if *version != "" {
		logrus.Info("Version: ", ReleaseVersion)
		os.Exit(0)
	}

	// Some simple validation
	if *pubSubTopic == "" {
		return nil, errors.New("PubSub topic is required")
	}

	if *contentTopic == "" {
		return nil, errors.New("Content topic is required")
	}

	if *msgPerSecond == 0 {
		return nil, errors.New("Number of messages per second is required")
	}

	if *msgSizeKb == 0 {
		return nil, errors.New("Message size in kilobytes is required")
	}

	if *bootstrapNode == "" {
		return nil, errors.New("Bootstrap node is required")
	}

	conf := &Config{
		PubSubTopic:   *pubSubTopic,
		ContentTopic:  *contentTopic,
		MsgPerSecond:  *msgPerSecond,
		MsgSizeKb:     *msgSizeKb,
		BootstrapNode: *bootstrapNode,
	}
	logConfig(conf)
	return conf, nil
}

func logConfig(cfg *Config) {
	logrus.WithFields(logrus.Fields{
		"PubSubTopic":   cfg.PubSubTopic,
		"ContentTopic":  cfg.ContentTopic,
		"MsgPerSecond":  cfg.MsgPerSecond,
		"MsgSizeKb":     cfg.MsgSizeKb,
		"BootstrapNode": cfg.BootstrapNode,
	}).Info("Cli Config:")
}
