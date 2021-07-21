package pb

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"strings"
)

func ContentToBytes(content proto.Message) ([]byte, error) {
	any, err := anypb.New(content)

	if err != nil {
		return nil, err
	}
	encodedcontent, err := proto.Marshal(any)
	return encodedcontent, err
}

func BytesToMessage(trxid string, content []byte) (proto.Message, string, error) {
	anyobj := &anypb.Any{}
	err := proto.Unmarshal(content, anyobj)
	if err != nil {
		return nil, "", fmt.Errorf("Unmarshal trx.Data id %s Err: %s", trxid, err)
	}
	var ctnobj proto.Message
	var typeurl string
	ctnobj, err = anyobj.UnmarshalNew()
	if err != nil { //old data pb.Object{} compatibility
		ctnobj = &Object{}
		err = proto.Unmarshal(content, ctnobj)
		if err != nil {
			return nil, "", fmt.Errorf("try old data compatibility Unmarshal %s Err: %s", trxid, err)
		} else {
			typeurl = "quorum.pb.Object"
		}
	} else {
		typeurl = strings.Replace(anyobj.TypeUrl, "type.googleapis.com/", "", 1)
	}
	return ctnobj, typeurl, nil
}
