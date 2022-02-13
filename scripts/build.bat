:: This .bat is just for windows.
:: Run it like:
::     D:\your-work-dir\quorum> .\scripts\build.bat
:: You'll get the quorum binary in the dirpath: D:\your-work-dir\quorum\dist

:: if you got an error when running this .bat:
::     cgo: exec gcc: exec: "gcc": executable file not found in %PATH%
:: maybe you need to install `mingw`  <https://sourceforge.net/projects/mingw-w64/files/latest/download>
:: remember to set `path`(系统环境变量) with the bin-path of `mingw` and then restart the shell.

set CGO_ENABLED=0

:: windows
go build -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" -o .\dist\windows_amd64\quorum.exe .\cmd\main.go

:: darwin
:: go build -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" -o .\dist\darwin_amd64\quorum .\cmd\main.go

:: linux
:: go build -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" -o .\dist\linux_amd64\quorum .\cmd\main.go
