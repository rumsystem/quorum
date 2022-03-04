package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type PSPing struct {
	Topic        *pubsub.Topic
	Subscription *pubsub.Subscription
	PeerId       peer.ID
	ps           *pubsub.PubSub
	ctx          context.Context
}

type PingResult struct {
	Seqnum  int32
	Req_at  int64
	Resp_at int64
}

var ping_log = logging.Logger("ping")

func NewPSPingService(ctx context.Context, ps *pubsub.PubSub, peerid peer.ID) *PSPing {
	psping := &PSPing{PeerId: peerid, ps: ps, ctx: ctx}
	return psping
}

func (p *PSPing) EnablePing() error {
	peerid := p.PeerId.Pretty()
	var err error
	topicid := fmt.Sprintf("PSPing:%s", peerid)
	p.Topic, err = p.ps.Join(topicid)
	if err != nil {
		ping_log.Infof("Enable PSPing channel <%s> failed", topicid)
		return err
	} else {
		ping_log.Infof("Enable PSPing channel <%s> done", topicid)
	}

	p.Subscription, err = p.Topic.Subscribe()
	if err != nil {
		ping_log.Fatalf("Subscribe PSPing channel <%s> failed", topicid)
		ping_log.Fatalf(err.Error())
		return err
	} else {
		ping_log.Infof("Subscribe PSPing channel <%s> done", topicid)
	}

	go p.handlePingRequest()
	return nil
}

func (p *PSPing) PingReq(dstpeerid string) ([10]int64, error) {
	result := [10]int64{}
	pingTimeout := time.Second * 5
	errCh := make(chan error, 1)
	timer := time.NewTimer(pingTimeout)
	dsttopicid := fmt.Sprintf("PSPing:%s", dstpeerid)
	var err error
	p.Topic, err = p.ps.Join(dsttopicid)
	if err != nil {
		ping_log.Errorf("Join PSPing dest channel <%s> failed:%s", dsttopicid, err.Error())
		return result, err
	}
	p.Subscription, err = p.Topic.Subscribe()
	if err != nil {
		ping_log.Errorf("Subscribe PSPing dest channel <%s> failed:%s", dsttopicid, err.Error())
		return result, err
	} else {
		ping_log.Infof("Subscribe PSPing dest channel <%s> done", dsttopicid)
	}

	defer func() {
		timer.Stop()
		close(errCh)
		p.Subscription.Cancel()
		err := p.Topic.Close()
		if err != nil {
			ping_log.Infof("Close PSPing Topic <%s> failed", dsttopicid)
		}
	}()
	if err != nil {
		ping_log.Infof("Join PSPing channel <%s> failed", dsttopicid)
		return result, err
	} else {
		ping_log.Infof("Join PSPing channel <%s> done", dsttopicid)
	}

	resultmap := make(map[[32]byte]*PingResult)

	for i := 0; i < 10; i++ {
		var payload [32]byte
		_, err := rand.Read(payload[0:32])
		pingobj := &quorumpb.PSPing{Seqnum: int32(i), IsResp: false, TimeStamp: time.Now().UnixNano(), Payload: payload[:]}
		bytes, err := proto.Marshal(pingobj)
		if err == nil {
			p.Topic.Publish(p.ctx, bytes)
			resultmap[payload] = &PingResult{Seqnum: pingobj.Seqnum, Req_at: pingobj.TimeStamp, Resp_at: 0}
		} else {
			ping_log.Errorf("Ping packet error <%s>", err)
		}
	}

	go p.handlePingResponse(&resultmap, errCh)
	select {
	case <-timer.C:
		ping_log.Error("PSPing timeout")
	case err := <-errCh:
		ping_log.Debugf("PSPing loop exit wit error: %s", err)
	}

	for _, v := range resultmap {
		if v.Seqnum < 10 {
			if v.Resp_at > 0 {
				result[v.Seqnum] = (v.Resp_at - v.Req_at) / int64(time.Millisecond)
			}
		}
	}
	return result, nil
}

func (p *PSPing) handlePingRequest() error {
	for {
		pingreqmsg, err := p.Subscription.Next(p.ctx)
		if err == nil {
			if pingreqmsg.ReceivedFrom != p.PeerId { //not me
				var pspingreq quorumpb.PSPing
				if err := proto.Unmarshal(pingreqmsg.Data, &pspingreq); err != nil {
					return err
				}
				pingobj := &quorumpb.PSPing{Seqnum: pspingreq.Seqnum, IsResp: true, TimeStamp: pspingreq.TimeStamp, Payload: pspingreq.Payload}
				bytes, err := proto.Marshal(pingobj)
				if err == nil {
					p.Topic.Publish(p.ctx, bytes)
				} else {
					ping_log.Errorf("Ping packet error <%s>", err)
				}
			}
		} else {
			ping_log.Errorf(err.Error())
			return err
		}
	}
}

func (p *PSPing) handlePingResponse(pingresult *map[[32]byte]*PingResult, errCh chan error) error {
	count := 0
	for {
		pingrespmsg, err := p.Subscription.Next(p.ctx)
		if err == nil {
			ping_log.Debugf("Ping packet recv from <%s>", p.PeerId)
			if pingrespmsg.ReceivedFrom != p.PeerId { //not me
				var pspingresp quorumpb.PSPing
				if err := proto.Unmarshal(pingrespmsg.Data, &pspingresp); err != nil {
					return err
				}
				if pspingresp.IsResp == true {
					var payload [32]byte
					copy(payload[:], pspingresp.Payload[0:32])
					_, ok := (*pingresult)[payload]
					if ok {
						(*pingresult)[payload].Resp_at = time.Now().UnixNano()
						count++
						if count == 10 {
							errCh <- nil
							return nil
						}
					}
				}
			}
		} else {
			ping_log.Error(err.Error())
			return err
		}
	}
}
