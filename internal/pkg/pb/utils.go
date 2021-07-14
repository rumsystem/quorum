package pb

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func ContentToBytes(content proto.Message) ([]byte, error) {
	any, err := anypb.New(content)

	if err != nil {
		return nil, err
	}
	encodedcontent, err := proto.Marshal(any)

	//var encodedcontent []byte
	//var err error
	//switch c := content.(type) {
	//case *Person:
	//case *Object:
	//	encodedcontent, err = proto.Marshal(c)
	//default:
	//	return nil, fmt.Errorf("unsupported type")
	//}
	return encodedcontent, err
}
