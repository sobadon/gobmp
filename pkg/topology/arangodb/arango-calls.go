package arangodb

import (
	"strconv"

	"github.com/golang/glog"
	"github.com/sbezverk/gobmp/pkg/message"
	"github.com/sbezverk/gobmp/pkg/srv6"
	_ "github.com/sbezverk/gobmp/pkg/topology/database"
)

const (
	l3prefix = "L3VPN_Prefix"
)

// L3VPNPrefix represents the database record structure for L3VPN Prefix collection
type L3VPNPrefix struct {
	Key       string          `json:"_key,omitempty"`
	ID        string          `json:"_id,omitempty"`
	Prefix    string          `json:"Prefix,omitempty"`
	PrefixLen int32           `json:"Length,omitempty"`
	IsIPv4    bool            `json:"IPv4"`
	OriginAS  int32           `json:"ASN,omitempty"`
	Nexthop   string          `json:"RouterID,omitempty"`
	Labels    []uint32        `json:"VPN_Label,omitempty"`
	RD        string          `json:"RD,omitempty"`
	ExtComm   []string        `json:"ExtComm,omitempty"`
	SRv6SID   *srv6.L3Service `json:"SRv6_SID,omitempty"`
}

// L3VPNRT defines route target record
type L3VPNRT struct {
	ID       string            `json:"_id,omitempty"`
	Key      string            `json:"_key,omitempty"`
	RT       string            `json:"RT,omitempty"`
	Prefixes map[string]string `json:"Prefixes,omitempty"`
}

func (a *arangoDB) l3vpnHandler(obj *message.L3VPNPrefix) {
	//	db := a.GetArangoDBInterface()
	if obj == nil {
		glog.Warning("L3 VPN Prefix object is nil")
		return
	}
	k := obj.VPNRD + "_" + obj.Prefix + "_" + strconv.Itoa(int(obj.PrefixLen))
	r := &L3VPNPrefix{
		Key:       k,
		ID:        l3prefix + "/" + k,
		Prefix:    obj.Prefix,
		PrefixLen: obj.PrefixLen,
		IsIPv4:    obj.IsIPv4,
		OriginAS:  obj.OriginAS,
		Nexthop:   obj.Nexthop,
		Labels:    obj.Labels,
		RD:        obj.VPNRD,
		ExtComm:   obj.BaseAttributes.ExtCommunityList,
	}

	if obj.PrefixSID != nil {
		r.SRv6SID = obj.PrefixSID.SRv6L3Service
	}

	glog.Infof("Resulting record: %+v", *r)
}