package data

import (
	"encoding/binary"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

const TRX_DATA_SIZE_LIMIT = 300 * 1024 //300Kb
var dataLog = logging.Logger("data")

func IsTrxDataWithinSizeLimit(data []byte) (bool, error) {
	size := binary.Size(data)
	if size > TRX_DATA_SIZE_LIMIT {
		e := fmt.Errorf("trx.Data size %dkb over %dkb", size/1024, TRX_DATA_SIZE_LIMIT/1024)
		dataLog.Warn(e)
		return false, e
	}

	return true, nil
}
