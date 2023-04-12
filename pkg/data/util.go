package data

import (
	"encoding/binary"
	"fmt"
)

const POST_OBJ_SIZE_LIMIT = 300 * 1024 //300Kb

func IsTrxWithinSizeLimit(content []byte) (bool, error) {
	if binary.Size(content) > POST_OBJ_SIZE_LIMIT {
		return false, fmt.Errorf("content size over %dKb", POST_OBJ_SIZE_LIMIT/1024)
	}

	return true, nil
}
