package main

import (
	"fmt"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"io/ioutil"
	"log"
	//"github.com/huo-ju/quorum/examples/pb/examplespb"
	"google.golang.org/protobuf/proto"
)

func main() {

	//{
	//  "@context": "https://www.w3.org/ns/activitystreams",
	//  "summary": "huoju created a note",
	//  "type": "Create",
	//  "quorumpb.BlockMessageactor": {
	//    "type": "Person",
	//    "name": "huoju"
	//  },
	//  "object": {
	//    "type": "Note",
	//    "name": "A Simple Note",
	//    "content": "This is a simple note by huoju"
	//  }
	//}
	//activity := &quorumpb.Activity{Type: "Create"}

	//	{
	//  "@context": "https://www.w3.org/ns/activitystreams",
	//  "summary": "Sally added a picture of her cat to her cat picture collection",
	//  "type": "Add",
	//  "actor": {
	//    "type": "Person",
	//    "name": "Sally"
	//  },
	//  "object": {
	//    "type": "Image",
	//    "name": "A picture of my cat",
	//    "url": "http://example.org/img/cat.png"
	//  },
	//  "origin": {
	//    "type": "Collection",
	//    "name": "Camera Roll"
	//  },
	//  "target": {
	//    "type": "Collection",
	//    "name": "My Cat Pictures"
	//  }
	//}

	//blockmsg := &quorumpb.BlockMessage{
	//    Type: quorumpb.BlockMessage_ASKNEXT,
	//    Value: "",
	//}
	//fmt.Println("===")
	//fmt.Println(blockmsg)
	//p := &examplespb.Person{
	//    Id:    1234,
	//    Name:  "John Doe",
	//    Email: "jdoe@example.com",
	//}

	//bodyBytes := []byte("{'ACTION'='POST_TO_GROUP', 'GROUP_ID'='test_group_id, " + "'PUBLISHER'='" + chain.GetContext().PeerId.Pretty() + "', 'CONTENT'='some test content'")
	//Add to group
	actor := &quorumpb.Object{Type: "Person", Name: "Huo Ju", Id: "my PeerId"}
	note := &quorumpb.Object{Content: "This is a simple note by huoju", Name: "A simple Node", Type: "Note"}
	grouptarget := &quorumpb.Object{Type: "Group", Id: "test_group_id"}
	addtogroupactivity := &quorumpb.Activity{Type: "Add", Object: note, Actor: actor, Target: grouptarget}
	out, err := proto.Marshal(addtogroupactivity)

	fmt.Println("testpb")
	//fmt.Println(p)
	// Write the new address book back to disk.
	fname := "testactivity.pb"
	if err != nil {
		log.Fatalln("Failed to encode address book:", err)
	}
	if err := ioutil.WriteFile(fname, out, 0644); err != nil {
		log.Fatalln("Failed to write address book:", err)
	}
	// Read the existing address book.
	in, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}
	np := &quorumpb.Object{}
	if err := proto.Unmarshal(in, np); err != nil {
		log.Fatalln("Failed to parse person:", err)
	}
	//if np.Type == quorumpb.BlockMessage_ASKHEAD {
	//	fmt.Println("ASKHEAD")
	//}
	fmt.Println(np)

}
