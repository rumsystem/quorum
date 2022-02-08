protoc -I=internal/pkg/pb --go_out=internal/pkg/pb internal/pkg/pb/chain.proto
mv internal/pkg/pb/github.com/rumsystem/quorum/internal/pkg/pb/chain.pb.go internal/pkg/pb/chain.pb.go
sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' internal/pkg/pb/chain.pb.go

protoc -I=internal/pkg/pb --go_out=internal/pkg/pb internal/pkg/pb/rumexchange.proto 
mv internal/pkg/pb/github.com/rumsystem/quorum/internal/pkg/pb/rumexchange.pb.go internal/pkg/pb/rumexchange.pb.go

protoc -I=internal/pkg/pb --go_out=internal/pkg/pb internal/pkg/pb/activity_stream.proto
mv internal/pkg/pb/github.com/rumsystem/quorum/internal/pkg/pb/activity_stream.pb.go internal/pkg/pb/activity_stream.pb.go
sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' internal/pkg/pb/activity_stream.pb.go
