Synthetic test
---

Run 3 adapter pods and stop the gRPC service on one of them (container still shows up as healthy). 
See if mixer recovers. 
While this is never expected to happen in production for an extended period of time, if we can make the system recover 
in this picture we will see fewer failures overall.

### Scenario 1: No sidecar on adapter, cluster IP (no envoy for mixer client)

- connections terminate, grpc client retries connection
- **eventually succeeds** with new IPTables route although there are still several blips

```
* rpc error: code = Unavailable desc = all SubConns are in TransientFailure, latest connection error: connection error: desc = "transport: Error while dialing dial tcp 100.68.29.172:4080: connect: connection refused"
2019-10-01T20:39:07.458081Z	info	newAttemptLocked failed!
2019-10-01T20:39:07.458105Z	warn	grpcAdapter	unable to connect to:/chaos.HandleChaosService/HandleChaos, chaos-adapter.chaos:4080, rpc error: code = Unavailable desc = all SubConns are in TransientFailure, latest connection error: connection error: desc = "transport: Error while dialing dial tcp 100.68.29.172:4080: connect: connection refused"
2019-10-01T20:39:07.458143Z	error	api	Check failed: performing check operation failed: 1 error occurred:
```

### Scenario 2: Sidecar on adapter, cluster IP (no envoy for mixer client)

- connection never terminates instead it is a RECVMSG error
- never succeeds because it never tries to reconnect

```
* rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:29:46.765148Z	info	RECVMSG rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:29:46.765169Z	warn	grpcAdapter	unable to connect to:/chaos.HandleChaosService/HandleChaos, chaos-adapter.chaos:4080, rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:29:46.765205Z	error	api	Check failed: performing check operation failed: 1 error occurred:
```

### Sidecar on adapter, Sidecar for mixer

- same as above

```
* rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:33:23.505862Z	info	RECVMSG rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:33:23.505602Z	info	RECVMSG rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:33:23.505886Z	warn	grpcAdapter	unable to connect to:/chaos.HandleChaosService/HandleChaos, localhost:15300, rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:33:23.505919Z	error	api	Check failed: performing check operation failed: 1 error occurred:
```

### No sidecar on adapter, sidecar for mixer

- same as above

```
* rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:33:23.505602Z	info	RECVMSG rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:33:23.505886Z	warn	grpcAdapter	unable to connect to:/chaos.HandleChaosService/HandleChaos, localhost:15300, rpc error: code = Unavailable desc = upstream connect error or disconnect/reset before headers. reset reason: connection failure
2019-10-01T19:33:23.505919Z	error	api	Check failed: performing check operation failed: 1 error occurred:
```

