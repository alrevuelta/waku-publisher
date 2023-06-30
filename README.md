# waku-publisher

Proof of concept:

```
go build
./main --pubsub-topic="my-ptopic"  --content-topic="my-ctopic" --msg-per-second=10 --msg-size-kb=10 --bootstrap-node="enr:xxx"
```

Optionally, a `private-key` can be passed to sign the messages (see [spec](https://rfc.vac.dev/spec/57/#dos-protection))

Docker images are available, last 7 digits of the commit: alrevuelta/waku-publisher:7342b2e
