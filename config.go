package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"flag"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	logrus "github.com/sirupsen/logrus"
)

type Config struct {
	PubSubTopic    string
	ContentTopic   string
	MsgPerSecond   uint64
	MsgSizeKb      uint64
	BootstrapNode  string
	PublishInvalid bool
	PrivateKey     *ecdsa.PrivateKey
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
	var privateKey = flag.String("private-key", "", "Key used to sign messages in configured pubsub topic")
	var publishInvalid = flag.Bool("publish-invalid", false, "Publish invalid messages to the configured pubsub topic (true/false). default: false")

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

	var pKey *ecdsa.PrivateKey
	var err error
	pKey = nil

	if *privateKey != "" {
		pKey, err = crypto.HexToECDSA(*privateKey)
		if err != nil {
			return nil, errors.New("wrong private key, couldn't parse it: " + err.Error())
		}
		publicKeyECDSA, ok := pKey.Public().(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("error casting public key to ECDSA: " + err.Error())
		}
		//address = crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
		pubKBytes := crypto.FromECDSAPub(publicKeyECDSA)
		logrus.Info("Configured public key: ", hex.EncodeToString(pubKBytes))
	} else {
		logrus.Info("No key was configured to sign messages with")
	}

	conf := &Config{
		PubSubTopic:    *pubSubTopic,
		ContentTopic:   *contentTopic,
		MsgPerSecond:   *msgPerSecond,
		MsgSizeKb:      *msgSizeKb,
		BootstrapNode:  *bootstrapNode,
		PublishInvalid: *publishInvalid,
		PrivateKey:     pKey,
	}
	logConfig(conf)
	return conf, nil
}

func logConfig(cfg *Config) {
	logrus.WithFields(logrus.Fields{
		"PubSubTopic":    cfg.PubSubTopic,
		"ContentTopic":   cfg.ContentTopic,
		"MsgPerSecond":   cfg.MsgPerSecond,
		"MsgSizeKb":      cfg.MsgSizeKb,
		"PublishInvalid": cfg.PublishInvalid,
		"BootstrapNode":  cfg.BootstrapNode,
	}).Info("Cli Config:")
}
