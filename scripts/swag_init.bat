:: This .bat is just for windows.
:: Run it like:
::     D:\your-work-dir\quorum> .\scripts\swag_init.bat
:: You'll get the new dir for api docs: D:\your-work-dir\quorum\docs

cd .\cmd
swag init -g main.go --parseDependency --parseInternal --parseDepth 2 --output=..\docs