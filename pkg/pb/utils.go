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

func BytesToMessageDefault(content []byte) (proto.Message, string, error) {
	anyobj := &anypb.Any{}
	var msg proto.Message
	if len(content) == 0 {
		return nil, "", fmt.Errorf("content must not be nil")
	}
	err := proto.Unmarshal(content, anyobj)
	if err != nil {
		return nil, "", err
	}

	msg, err = anyobj.UnmarshalNew()
	if err != nil {
		return nil, "", err
	}
	typeurl := strings.Replace(anyobj.TypeUrl, "type.googleapis.com/", "", 1)
	return msg, typeurl, err
}

func BytesToMessage(trxid string, content []byte) (proto.Message, string, error) {
	anyobj := &anypb.Any{}

	var ctnobj proto.Message
	var typeurl string

	if len(content) == 0 {
		ctnobj = &Object{}
		typeurl = "quorum.pb.Object"
		return ctnobj, typeurl, nil
	}

	err := proto.Unmarshal(content, anyobj)
	if err != nil {
		return nil, "", fmt.Errorf("Unmarshal trx.Data id %s Err: %s", trxid, err)
	}
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
