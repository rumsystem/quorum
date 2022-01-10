# rumcli

A termnal client for quorum.

## Prepare

The rumcli needs an API server running first, all operations will go through the API server.

Use following command to start a local api server with a remote peer.

```bash
quorum -peername chux0519_peer -listen /ip4/0.0.0.0/tcp/7002  -apilisten :8002 -peer /ip4/167.114.61.179/tcp/10666/p2p/16Uiu2HAmE7TYnrYtC6tAsWvz4EUGQke1wsdrkw2kt8GFv6brfHFw  -debug true
```

[The quorum project](https://github.com/rumsystem/quorum)

## Start

Just run rumcli.

The rumcli use serveral commands to interact with the API server.

In rumcli, press `Space` to bring up the command prompt, then

- `/connect` connect to an API server, for example, run 

    > /connect 127.0.0.1:8002

    It will connect to your API server, after that, you will see the status of your server and so on.

- `/join` join into a group, for example, run

    > /join @/home/user/seed.json

     It will use the `/home/user/seed.json` and try to join that group. 

     You can use plain text without the `@` prefix as well, like 
    > /join $plain-json-text

- `/send` to send content to group, for example, run

    > /send hello, world

    It will send "hello, world" to current group.

- `/group.create` to new a group, for example, run

    > /group.create mygroup1

    It will create a group, and save the seed to your /tmp/xxxx

- `/group.leave` to leave a group

- `/group.delete` to delete a group (only group owner can do this)

- `/config.reload` to reload config file

- `/config.save` to save config file manually

- Press `?` to show more

## Shortcuts

By default, the rumcli will use vim like keybindings.

- `?` to show help
- Shift + H/J/K/L to move between widgets
- Tab / Shift+Tab to naviagate items in widgets
- Enter
  - In GroupList View, it will switch to the group you select
  - In Content View, it will fetch the detail about the content you select
  - In Root View, it will focus on Content View

In each widget, h/j/k/l can scoll the content.

In GroupList View, press (a-z) to quick-switch between groups.

In ContentView

- press Shift + F to follow someone, Shift + U to unfollow.
- Shift + 1 to show all contents, Shift + 2 to show following contens, Shift + 3 to show your own contents

## Configs

rumcli will load your config on starting. Default paths are different on different operating systems.

Linux: `~/.config/rumcli/config.toml`
OSX: `~/Library/Application Support/rumcli/config.toml`
Windows: `C:\Users\xxxx\AppData\Local\rumcli\config.toml`

Fill the `ServerSSLCertificate` like `ServerSSLCertificate = "/home/xxx/quorum/certs/server.crt"` before using to avoid https error.

Notice: On windows you have to use the escaped string, for example `C:\\Users\\xxxx\\repos\\quorum\\certs\\server.crt`

## Debug

```bash
$> go build -gcflags=all="-N -l" rumcli.go
$> ./rumcli
$> dlv attach $(pgrep rumcli)
```
