package metric

import (
	"encoding/binary"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"google.golang.org/protobuf/proto"
)

var logger = logging.Logger("metric")

func GetBinarySize(v interface{}) uint {
	size := binary.Size(v)
	if size < 0 {
		size = 0
		logger.Errorf("get binary.Size(%+v) failed", v)
	}

	return uint(size)
}

func GetProtoSize(v proto.Message) uint {
	size := proto.Size(v)

	if size < 0 {
		size = 0
		logger.Errorf("get proto.Size(%+v) failed", v)
	}

	return uint(size)
}
