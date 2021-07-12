package pb

import (
	"fmt"
	"google.golang.org/protobuf/proto"
)

func ContentToBytes(content interface{}) ([]byte, error) {
	var encodedcontent []byte
	var err error
	switch c := content.(type) {
	case *Person:
	case *Object:
		encodedcontent, err = proto.Marshal(c)
	default:
		return nil, fmt.Errorf("unsupported type")
	}
	return encodedcontent, err
}
