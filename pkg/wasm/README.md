## wasm

quorum 可以被编译到 wasm 从而在浏览器中运行，大部分功能和 native 的版本一致，主要区别在于文件系统操作和 socket 操作。
主要把这些操作抽象到 interface，想要 port 到浏览器就会比较容易。

简单讲目前，badger 和 socket 操作需要抽象。

## 文件系统

native 中可以调用操作系统提供的各种操作文件的 API，可以使用如 badger 之类的基于文件系统的数据库。

浏览器中没有这些功能，但是浏览器中提供 indexeddb，可以基于它完成 native 版本中相应的业务逻辑功能。

1. 对于 badger，相应的 interface 可以在 internal/pkg/storage/storage.go 中查看
2. 对于 keystore，相应的 interface 在internal/pkg/crypto/keystore.go 中查看

对于不同的平台，实现相同的 interface 即可，例如 `keystore_native.go`, `keystore_browser.go`，不同平台的代码，使用 build flag 保护源码即可

比如 

```
//go:build js && wasm
// +build js,wasm
```

说明该代码是为浏览器准备的。

## socket

浏览器中没办法使用 socket 的 API，native 平台可以监听端口，操控各种文件描述符，但是浏览器中只能使用浏览器的 API （http 和 websocket），因此对于 quorum 而言，涉及到端口监听的操作是不支持的，另外网络请求也只能通过 http 之上的协议进行，因此目前 quorum 的浏览器版本只能连接支持 websocket 的节点。

## 编译和 demo

可以参考 cmd/wasm/README