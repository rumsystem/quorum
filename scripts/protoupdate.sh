protoc -I=pkg/pb --go_out=pkg/pb pkg/pb/chain.proto
mv pkg/pb/github.com/rumsystem/quorum/pkg/pb/chain.pb.go pkg/pb/chain.pb.go
sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' pkg/pb/chain.pb.go

protoc -I=pkg/pb --go_out=pkg/pb pkg/pb/rumexchange.proto 
mv pkg/pb/github.com/rumsystem/quorum/pkg/pb/rumexchange.pb.go pkg/pb/rumexchange.pb.go

protoc -I=pkg/pb --go_out=pkg/pb pkg/pb/activity_stream.proto
mv pkg/pb/github.com/rumsystem/quorum/pkg/pb/activity_stream.pb.go pkg/pb/activity_stream.pb.go
sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' pkg/pb/activity_stream.pb.go
