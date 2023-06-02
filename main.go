package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	log "github.com/sirupsen/logrus"
	logrus "github.com/sirupsen/logrus"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap/zapcore"
)

var msgSent = 0

//var log = utils.Logger().Named("basic2")

func newConnManager(lo int, hi int, opts ...connmgr.Option) *connmgr.BasicConnMgr {
	mgr, err := connmgr.NewConnManager(lo, hi, opts...)
	if err != nil {
		panic("could not create ConnManager: " + err.Error())
	}
	return mgr
}

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	cfg, err := NewCliConfig()
	if err != nil {
		log.Fatal("error parsing config: ", err)
	}

	hostAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	key, err := randomHex(32)
	if err != nil {
		log.Error("Could not generate random key: ", err)
		return
	}
	prvKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Error("Could not convert hex into ecdsa key: ", err)
		return
	}

	ctx := context.Background()

	customNodes := []*enode.Node{
		enode.MustParse(cfg.BootstrapNode),
		//enode.MustParse("enr:-M-4QOth4Dg45mYtbfMf3YZOLaAVrQuNWEyb-rahElFJHeBkCTXe0AMPXO_XtT05UK3_v6nEfQOLWaVGt6WUsM_BpA0BgmlkgnY0gmlwhI_G-a6KbXVsdGlhZGRyc7EALzYobm9kZS0wMS5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1c2ltLm5ldAYBu94DiXNlY3AyNTZrMaECoVyonsTGEQvVioM562Q1fjzTb_vKD152PPIdsV7sM6SDdGNwgnZfg3VkcIIjKIV3YWt1Mg8"),
		//enode.MustParse("enr:-M-4QHL_casP1Jy4KntHNWT3p1XkPxm1BJSxDi7KucSqZ2PgT97d4xEQ4cJx-bgw0SRu-nO4y5k0jTQN4AH7utodtZMBgmlkgnY0gmlwhKEj9HmKbXVsdGlhZGRyc7EALzYobm9kZS0wMi5kby1hbXMzLnN0YXR1cy5wcm9kLnN0YXR1c2ltLm5ldAYBu94DiXNlY3AyNTZrMaED1AYI2Ox27DnSqf2qoih5M2fNpHFq-OzJ3thREEApdiiDdGNwgnZfg3VkcIIjKIV3YWt1Mg8"),
	}
	log.Info("DefaultLibP2POptions len: ", len(node.DefaultLibP2POptions))
	if len(node.DefaultLibP2POptions) != 5 {
		log.Fatal("DefaultLibP2POptions has changed, please update this code")
	}
	// Very dirty way to inject a new ConnManager
	// TODO: This should keep the max amount of peers to 4 but the manager has a hard time doing so
	// it stays around 6 or so. Need to investigate
	node.DefaultLibP2POptions[4] = libp2p.ConnectionManager(newConnManager(1, 4, connmgr.WithGracePeriod(0)))

	wakuNode, err := node.New(
		node.WithPrivateKey(prvKey),
		node.WithHostAddress(hostAddr),
		node.WithNTP(),
		node.WithWakuRelay(),
		node.WithDiscoveryV5(8000, customNodes, true),
		node.WithDiscoverParams(30),
		node.WithLogLevel(zapcore.Level(zapcore.ErrorLevel)),
	)
	if err != nil {
		log.Error("Error creating wakunode: ", err)
		return
	}

	if err := wakuNode.Start(ctx); err != nil {
		log.Error("Error starting wakunode: ", err)
		return
	}

	err = wakuNode.DiscV5().Start(ctx)
	if err != nil {
		log.Fatal("Error starting discovery: ", err)
	}

	go logPeriodicInfo(wakuNode)
	go writeLoop(ctx, wakuNode, cfg)
	go runEverySecond(wakuNode, cfg)

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\n\n\nReceived signal, shutting down...")

	// shut the node down
	wakuNode.Stop()
}

func logPeriodicInfo(wakuNode *node.WakuNode) {
	for {
		fmt.Println("Connected peers", wakuNode.PeerCount(), " msg sent: ", msgSent)
		time.Sleep(10 * time.Second)
	}
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func runEverySecond(wakuNode *node.WakuNode, cfg *Config) {
	ticker := time.NewTicker(1 * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			start := time.Now().UnixNano() / int64(time.Millisecond)

			// Not very fancy. messages are not evenly distributed over time
			// eg 5 msg per second are send very quickly and the remaning time to complete
			// the second is idle.
			for i := 0; i < int(cfg.MsgPerSecond); i++ {
				write(context.Background(), wakuNode, cfg)
				msgSent++
			}
			end := time.Now().UnixNano() / int64(time.Millisecond)

			diff := end - start
			fmt.Println("Duration(ms):", diff)
			// do something if message rate is greater than what can be handled
			if diff > 1000 {
				fmt.Println("Warning: took more than 1 second")
			}
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func PublishRawToTopic(ctx context.Context, w *relay.WakuRelay, rawData []byte, topic string) {
	// Publish a `WakuMessage` to a PubSub topic.
	err := w.PubSub().Publish(topic, rawData)
	if err != nil {
		logrus.Error("Error publishing message: ", err)
	}
}

func write(
	ctx context.Context,
	wakuNode *node.WakuNode,
	cfg *Config) {

	var version uint32 = 0
	var timestamp int64 = utils.GetUnixEpoch(wakuNode.Timesource())

	randomPayload := make([]byte, cfg.MsgSizeKb*1000)
	rand.Read(randomPayload)

	p := new(payload.Payload)
	//p.Data = []byte(wakuNode.ID() + ": " + msgContent)
	p.Data = []byte(randomPayload)
	//p.Key = &payload.KeyInfo{Kind: payload.None}

	//fmt.Println("Payload size: ", len(p.Data), " bytes")

	payload, err := p.Encode(version)
	if err != nil {
		log.Fatal("Error encoding the payload: ", err)
		return
	}

	//log.Info("Message payload in kBytes: ", len(p.Data)/1000)

	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      version,
		ContentTopic: cfg.ContentTopic,
		Timestamp:    timestamp,
	}

	/*
		msgMarshal, err := msg.Marshal()
		if err != nil {
			log.Error("Error marshalling message: ", err)
		}
		fmt.Println("msg size: ", len(msgMarshal), " bytes")*/

	//_, err = wakuNode.Relay().Publish(ctx, msg)
	//wakuNode.Relay().PublishToTopic(ctx, msg, relay.DefaultWakuTopic)

	if cfg.PublishInvalid {
		// Publish invalid messages to the topic, just random bytes
		PublishRawToTopic(ctx, wakuNode.Relay(), msg.Payload, cfg.PubSubTopic)
	} else {
		// Sign only if a key was configured
		if cfg.PrivateKey != nil {
			err = relay.SignMessage(cfg.PrivateKey, cfg.PubSubTopic, msg)
			if err != nil {
				log.Fatal("Error signing message: ", err)
			}
		}
		_, err = wakuNode.Relay().PublishToTopic(ctx, msg, cfg.PubSubTopic)
		if err != nil {
			log.Error("Error sending a message: ", err)
		}
	}
}

func writeLoop(ctx context.Context, wakuNode *node.WakuNode, cfg *Config) {
	for {
		time.Sleep(2 * time.Second)
		write(ctx, wakuNode, cfg)
		msgSent++
	}
}
