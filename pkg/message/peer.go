package message

import (
	"crypto/md5"
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/sbezverk/gobmp/pkg/bmp"
)

func (p *producer) producePeerMessage(op int, msg bmp.Message) {
	if msg.PeerHeader == nil {
		glog.Errorf("perPeerHeader is missing, cannot construct PeerStateChange message")
		return
	}
	action := "add"
	if op == peerDown {
		action = "del"
	}

	var m PeerStateChange
	if op == peerUP {
		peerUpMsg, ok := msg.Payload.(*bmp.PeerUpMessage)
		if !ok {
			glog.Errorf("got invalid Payload type in bmp.Message %+v", msg.Payload)
			return
		}
		m = PeerStateChange{
			Action:         action,
			RemoteASN:      msg.PeerHeader.PeerAS,
			PeerRD:         msg.PeerHeader.GetPeerDistinguisherString(),
			RemotePort:     int(peerUpMsg.RemotePort),
			Timestamp:      msg.PeerHeader.GetPeerTimestamp(),
			LocalPort:      int(peerUpMsg.LocalPort),
			AdvHolddown:    int(peerUpMsg.SentOpen.HoldTime),
			RemoteHolddown: int(peerUpMsg.ReceivedOpen.HoldTime),
		}
		if msg.PeerHeader.FlagV {
			m.IsIPv4 = false
			m.RemoteIP = net.IP(msg.PeerHeader.PeerAddress).To16().String()
			m.LocalIP = net.IP(peerUpMsg.LocalAddress).To16().String()
			m.RemoteBGPID = net.IP(msg.PeerHeader.PeerBGPID).To16().String()
			m.LocalBGPID = net.IP(peerUpMsg.SentOpen.BGPID).To16().String()
		} else {
			m.IsIPv4 = true
			m.RemoteIP = net.IP(msg.PeerHeader.PeerAddress[12:]).To4().String()
			m.LocalIP = net.IP(peerUpMsg.LocalAddress[12:]).To4().String()
			m.RemoteBGPID = net.IP(msg.PeerHeader.PeerBGPID).To4().String()
			m.LocalBGPID = net.IP(peerUpMsg.SentOpen.BGPID).To4().String()
		}
		// Saving local bgp speaker identities.
		p.speakerIP = m.LocalIP
		p.speakerHash = fmt.Sprintf("%x", md5.Sum([]byte(p.speakerIP)))
		m.RouterIP = p.speakerIP
		m.RouterHash = p.speakerHash

		m.LocalASN = uint32(peerUpMsg.SentOpen.MyAS)
		if lasn, ok := peerUpMsg.SentOpen.Is4BytesASCapable(); ok {
			// Local BGP speaker is 4 bytes AS capable
			m.LocalASN = lasn
		}
		p.addPathCapable = make(map[int]bool)
		// Check if local router advertises AddPath Send/Receive for any AFI/SAFI,
		// if map comes back empty no further AddPath Capability is needed
		if lAddPath := peerUpMsg.SentOpen.AddPathCapability(); len(lAddPath) != 0 {
			// Check if remote router advertises AddPath Send/Receive for any AFI/SAFI,
			// if map comes back empty no further AddPath Capability is needed
			if rAddPath := peerUpMsg.ReceivedOpen.AddPathCapability(); len(rAddPath) != 0 {
				for k := range lAddPath {
					// Enable AddPath only for AFI/SAFI types existing in both local and remote maps
					if _, ok := rAddPath[k]; ok {
						// AFI/SAFI type exists in both maps, which means both peers support Send/Receive of AddPath
						p.addPathCapable[k] = true
					}
				}
			}
		}
		m.AdvCapabilities = peerUpMsg.SentOpen.GetCapabilities()
		m.RcvCapabilities = peerUpMsg.ReceivedOpen.GetCapabilities()
	} else {
		peerDownMsg, ok := msg.Payload.(*bmp.PeerDownMessage)
		if !ok {
			glog.Errorf("got invalid Payload type in bmp.Message")
			return
		}
		m = PeerStateChange{
			Action:     "down",
			RouterIP:   p.speakerIP,
			RouterHash: p.speakerHash,
			BMPReason:  int(peerDownMsg.Reason),
			RemoteASN:  msg.PeerHeader.PeerAS,
			PeerRD:     msg.PeerHeader.GetPeerDistinguisherString(),
			Timestamp:  msg.PeerHeader.GetPeerTimestamp(),
		}
		if msg.PeerHeader.FlagV {
			m.IsIPv4 = false
			m.RemoteIP = net.IP(msg.PeerHeader.PeerAddress).To16().String()
			m.RemoteBGPID = net.IP(msg.PeerHeader.PeerBGPID).To16().String()
		} else {
			m.IsIPv4 = true
			m.RemoteIP = net.IP(msg.PeerHeader.PeerAddress[12:]).To4().String()
			m.RemoteBGPID = net.IP(msg.PeerHeader.PeerBGPID).To4().String()
		}
		m.InfoData = make([]byte, len(peerDownMsg.Data))
		copy(m.InfoData, peerDownMsg.Data)

	}
	if err := p.marshalAndPublish(&m, bmp.PeerStateChangeMsg, []byte(m.RouterHash), false); err != nil {
		glog.Errorf("failed to process peer message with error: %+v", err)
		return
	}
}
