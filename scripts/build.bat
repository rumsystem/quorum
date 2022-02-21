:: This .bat is just for windows.

:: `build.sh` is always your first choice. 
:: When you need some help running  in windows system computer, you can try this .bat file.


:: Run it like:
::     D:\your-work-dir\quorum> .\scripts\build.bat
:: You'll get the quorum binary in the dirpath: D:\your-work-dir\quorum\dist

:: If you got an error when running this .bat:
::     cgo: exec gcc: exec: "gcc": executable file not found in %PATH%
:: maybe you need to install `mingw`  <https://sourceforge.net/projects/mingw-w64/files/latest/download>
:: Remember to set `path`(系统环境变量) with the bin-path of `mingw` and then restart the shell.

:: If you got an error when running this .bat:
::     # command-line-arguments
::     usage: link [options] main.o 
::     ...
:: Try to run the command lines directly in your shell. Just like:
::     D:\your-work-dir\quorum> set CGO_ENABLED=0
::     D:\your-work-dir\quorum> set GOOS=windows
::     D:\your-work-dir\quorum> set GOARCH=amd64
::     D:\your-work-dir\quorum> go build -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" -o dist\windows_amd64\quorum_win.exe cmd\main.go

:: windows
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" -o dist\windows_amd64\quorum_win.exe cmd\main.go


:: darwin
:: set CGO_ENABLED=0
:: set GOOS=darwin
:: set GOARCH=amd64
:: go build -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" -o .\dist\darwin_amd64\quorum .\cmd\main.go

:: linux
:: set CGO_ENABLED=0
:: set GOOS=linux
:: set GOARCH=amd64
:: go build -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" -o .\dist\linux_amd64\quorum .\cmd\main.go
