package storage

import (
    "encoding/json"
)

type Message struct{
    ID string
    Status string
}

type TextMessage struct{
    Message
    Content string
}

func (msg *TextMessage) ToJson() ([]byte, error) {
    return json.Marshal(struct {
		    ID string `json:"id"`
		    Status string `json:"status"`
            Content string `json:"content"`
	    }{
            ID: msg.Message.ID,
            Status: msg.Message.Status,
            Content: msg.Content,
	    })
}

func NewTextMessage(jsonData []byte) (*TextMessage, error){
    var data map[string]interface{}
    err := json.Unmarshal([]byte(jsonData), &data)
    if err != nil{
        return nil, err
    }
    msg := &Message{}
    textmsg := &TextMessage{}
    if id, ok := data["id"].(string); ok {
        msg.ID = id
    }
    if status , ok := data["status"].(string); ok {
        msg.Status = status
    }
    if content, ok := data["content"].(string); ok {
       textmsg.Content = content
    }
    textmsg.Message = *msg
    return textmsg, nil
}
