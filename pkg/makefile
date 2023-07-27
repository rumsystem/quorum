#PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
#PROTOC = $(shell which protoc)

compile: chain.proto activity_stream.proto rumexchange.proto

chain.proto:
	protoc -I=pb --go_out=pb pb/chain.proto
	mv pb/github.com/rumsystem/quorum/pkg/pb/chain.pb.go pb/chain.pb.go
	sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' pb/chain.pb.go

activity_stream.proto:
	protoc -I=pb --go_out=pb pb/activity_stream.proto
	mv pb/github.com/rumsystem/quorum/pkg/pb/activity_stream.pb.go pb/activity_stream.pb.go
	sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' pb/activity_stream.pb.go

rumexchange.proto:
	protoc -I=pb --go_out=pb pb/rumexchange.proto 
	mv pb/github.com/rumsystem/quorum/pkg/pb/rumexchange.pb.go pb/rumexchange.pb.go
	sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' pb/rumexchange.pb.go

build: compile

buildall: compile

all: compile
