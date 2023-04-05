package proxy

type ClientHookReceiveConnection func(proxyID, connID uint64)
type ClientHookBeforeConnectLocal func(proxyID, connID uint64, localAddr string)
type ClientHookAfterConnectLocal func(proxyID, connID uint64, localAddr string)
type ClientHookConnectionReset func(proxyID, connID uint64)
