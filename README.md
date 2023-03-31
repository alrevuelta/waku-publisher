# waku-publisher

Proof of concept:

```
go build
./main --pubsub-topic="my-ptopic"  --content-topic="my-ctopic" --msg-per-second=10 --msg-size-kb=10 --bootstrap-node="enr:xxx"
```