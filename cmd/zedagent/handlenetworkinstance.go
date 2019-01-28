// Copyright (c) 2018,2019 Zededa, Inc.
// All rights reserved.

// Handle NetworkInstanceStatus from zedrouter

package zedagent

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	log "github.com/sirupsen/logrus"
	"github.com/zededa/api/zmet"
	"github.com/zededa/go-provision/cast"
	"github.com/zededa/go-provision/types"
	"strings"
)

func handleNetworkInstanceModify(ctxArg interface{}, key string, statusArg interface{}) {
	log.Infof("handleNetworkInstanceStatusModify(%s)\n", key)
	ctx := ctxArg.(*zedagentContext)
	status := cast.CastNetworkInstanceStatus(statusArg)
	if status.Key() != key {
		log.Errorf("handleNetworkInstanceModify key/UUID mismatch %s vs %s; ignored %+v\n", key, status.Key(), status)
		return
	}
	if !status.ErrorTime.IsZero() {
		log.Errorf("Received NetworkInstance error %s\n",
			status.Error)
	}
	switch status.Type {
	case types.NetworkInstanceTypeMesh: // XXX any subtype?
		handleNetworkLispInstanceStatusModify(ctx, status)
	case types.NetworkInstanceTypeCloud:
		handleNetworkVpnInstanceStatusModify(ctx, status)
	default:
	}
	log.Infof("handleNetworkInstanceModify(%s) done\n", key)
}

func handleNetworkInstanceDelete(ctxArg interface{}, key string,
	statusArg interface{}) {

	log.Infof("handleNetworkInstanceDelete(%s)\n", key)
	status := cast.CastNetworkInstanceStatus(statusArg)
	if status.Key() != key {
		log.Errorf("handleNetworkInstanceDelete key/UUID mismatch %s vs %s; ignored %+v\n",
			key, status.Key(), status)
		return
	}
	ctx := ctxArg.(*zedagentContext)
	switch status.Type {
	case types.NetworkInstanceTypeMesh: // XXX any subtype?
		handleNetworkLispInstanceStatusDelete(ctx, status)
	case types.NetworkInstanceTypeCloud:
		handleNetworkVpnInstanceStatusDelete(ctx, status)
	default:
	}
	log.Infof("handleNetworkInstanceDelete(%s) done\n", key)
}

func handleNetworkVpnInstanceStatusModify(ctx *zedagentContext,
	status types.NetworkInstanceStatus) {
	prepareVpnInstanceInfoMsg(ctx, status, false)
}

func handleNetworkVpnInstanceStatusDelete(ctx *zedagentContext, status types.NetworkInstanceStatus) {
	prepareVpnInstanceInfoMsg(ctx, status, true)
}

func prepareAndPublishLispInstanceInfoMsg(ctx *zedagentContext,
	status types.NetworkInstanceStatus, deleted bool) {

	infoMsg := &zmet.ZInfoMsg{}
	infoType := new(zmet.ZInfoTypes)
	*infoType = zmet.ZInfoTypes_ZiNetworkInstance
	infoMsg.DevId = *proto.String(zcdevUUID.String())
	infoMsg.Ztype = *infoType

	uuid := status.Key()
	info := new(zmet.ZInfoNetworkInstance)
	info.NetworkID = uuid
	info.NetworkVersion = status.UUIDandVersion.Version
	info.Displayname = status.DisplayName
	info.InstType = uint32(status.Type)

	// Always need a lispinfo to satisfy oneof
	lispInfo := new(zmet.ZInfoLisp)
	if deleted {
		// XXX When a network instance is deleted it is ideal to
		// send a flag such as deleted/gone inside
		// ZInfoNetworkInstance message. Having a separate flag
		// (indicating deletion) would make is explicit
		// and easy for the cloud process.
		info.Activated = false
	} else {
		info.Activated = status.Activated

		info.BridgeNum = uint32(status.BridgeNum)
		info.BridgeName = status.BridgeName
		info.BridgeIPAddr = status.BridgeIPAddr

		for mac, ip := range status.IPAssignments {
			assignment := new(zmet.ZmetIPAssignmentEntry)
			assignment.MacAddress = mac
			assignment.IpAddress = append(assignment.IpAddress, ip.String())
			info.IpAssignments = append(info.IpAssignments,
				assignment)
		}
		for _, s := range status.BridgeIPSets {
			info.BridgeIPSets = append(info.BridgeIPSets, s)
		}
		for _, v := range status.Vifs {
			vi := new(zmet.ZmetVifInfo)
			vi.VifName = v.Name
			vi.MacAddress = v.MacAddr
			vi.AppID = v.AppID.String()
			info.Vifs = append(info.Vifs, vi)
		}
		info.Ipv4Eid = status.Ipv4Eid

		// For now we just send an empty lispInfo to indicate deletion to cloud.
		// It can't be omitted since protobuf requires something to satisfy
		// the oneof.

		lispStatus := status.LispInfoStatus
		if lispStatus != nil {
			fillLispInfo(lispInfo, lispStatus)
		}
	}

	info.InfoContent = new(zmet.ZInfoNetworkInstance_Linfo)
	if x, ok := info.GetInfoContent().(*zmet.ZInfoNetworkInstance_Linfo); ok {
		x.Linfo = lispInfo
	}

	infoMsg.InfoContent = new(zmet.ZInfoMsg_Niinfo)
	if x, ok := infoMsg.GetInfoContent().(*zmet.ZInfoMsg_Niinfo); ok {
		x.Niinfo = info
	}
	log.Debugf("Publish LispInfo message to zedcloud: %v\n", infoMsg)
	publishInfo(ctx, uuid, infoMsg)
}

func fillLispInfo(lispInfo *zmet.ZInfoLisp, lispStatus *types.LispInfoStatus) {

	lispInfo.ItrCryptoPort = lispStatus.ItrCryptoPort
	lispInfo.EtrNatPort = lispStatus.EtrNatPort
	for _, intf := range lispStatus.Interfaces {
		lispInfo.Interfaces = append(lispInfo.Interfaces, intf)
	}

	// Copy ITR database map entries
	for _, dbMap := range lispStatus.DatabaseMaps {
		dbMapEntry := &zmet.DatabaseMap{
			IID: dbMap.IID,
		}

		for _, mapEntry := range dbMap.MapCacheEntries {
			mapCacheEntry := &zmet.MapCacheEntry{
				EID: mapEntry.EID.String(),
			}

			for _, rloc := range mapEntry.Rlocs {
				rlocEntry := &zmet.RlocState{
					Rloc:      rloc.Rloc.String(),
					Reachable: rloc.Reachable,
				}
				mapCacheEntry.Rlocs = append(mapCacheEntry.Rlocs,
					rlocEntry)
			}
			dbMapEntry.MapCacheEntries = append(dbMapEntry.MapCacheEntries,
				mapCacheEntry)
		}
		lispInfo.DatabaseMaps = append(lispInfo.DatabaseMaps,
			dbMapEntry)
	}

	// Copy ETR decap entries
	for _, decapKey := range lispStatus.DecapKeys {
		decap := &zmet.DecapKey{
			Rloc:     decapKey.Rloc.String(),
			Port:     decapKey.Port,
			KeyCount: decapKey.KeyCount,
		}
		lispInfo.DecapKeys = append(lispInfo.DecapKeys, decap)
	}
}

func handleNetworkLispInstanceStatusModify(ctx *zedagentContext, status types.NetworkInstanceStatus) {
	prepareAndPublishLispInstanceInfoMsg(ctx, status, false)
}

func handleNetworkLispInstanceStatusDelete(ctx *zedagentContext, status types.NetworkInstanceStatus) {
	prepareAndPublishLispInstanceInfoMsg(ctx, status, true)
}

func prepareVpnInstanceInfoMsg(ctx *zedagentContext, status types.NetworkInstanceStatus, delete bool) {
	if status.VpnStatus == nil {
		return
	}
	infoMsg := &zmet.ZInfoMsg{}
	infoType := new(zmet.ZInfoTypes)
	*infoType = zmet.ZInfoTypes_ZiNetworkInstance
	infoMsg.DevId = *proto.String(zcdevUUID.String())
	infoMsg.Ztype = *infoType

	uuid := status.Key()
	vpnStatus := status.VpnStatus
	info := new(zmet.ZInfoNetworkInstance)
	info.NetworkID = uuid
	info.NetworkVersion = status.UUIDandVersion.Version
	info.Displayname = status.DisplayName
	info.InstType = uint32(status.Type)
	info.Activated = status.Activated
	info.SoftwareList = new(zmet.ZInfoSW)
	info.SoftwareList.SwVersion = vpnStatus.Version
	info.Activated = status.Activated
	upTime, _ := ptypes.TimestampProto(vpnStatus.UpTime)
	info.UpTimeStamp = upTime

	if !status.ErrorTime.IsZero() {
		errInfo := new(zmet.ErrorInfo)
		errInfo.Description = status.Error
		errTime, _ := ptypes.TimestampProto(status.ErrorTime)
		errInfo.Timestamp = errTime
		info.NetworkErr = append(info.NetworkErr, errInfo)
	}

	vpnInfo := new(zmet.ZInfoVpn)
	vpnInfo.PolicyBased = vpnStatus.PolicyBased
	listeningIpAddrs := strings.Split(vpnStatus.IpAddrs, " ")
	vpnInfo.ListeningIpAddrs = make([]string, len(listeningIpAddrs))
	for idx, ipAddr := range listeningIpAddrs {
		vpnInfo.ListeningIpAddrs[idx] = ipAddr
	}

	totalConnCount := len(vpnStatus.StaleVpnConns) + len(vpnStatus.ActiveVpnConns)

	if totalConnCount == 0 {
		info.InfoContent = new(zmet.ZInfoNetworkInstance_Vinfo)
		if x, ok := info.GetInfoContent().(*zmet.ZInfoNetworkInstance_Vinfo); ok {
			x.Vinfo = vpnInfo
		}

		// prapare the final stuff
		infoMsg.InfoContent = new(zmet.ZInfoMsg_Niinfo)
		if x, ok := infoMsg.GetInfoContent().(*zmet.ZInfoMsg_Niinfo); ok {
			x.Niinfo = info
		}
		publishInfo(ctx, uuid, infoMsg)
		return
	}

	vpnInfo.Conn = make([]*zmet.ZInfoVpnConn, totalConnCount)
	// stale connections
	connIdx := 0
	for _, vpnConn := range vpnStatus.StaleVpnConns {
		vpnConnInfo := publishVpnConnection(vpnInfo, vpnConn)
		if vpnConnInfo != nil {
			vpnInfo.Conn[connIdx] = vpnConnInfo
			connIdx++
		}
	}

	// active connections
	for _, vpnConn := range vpnStatus.ActiveVpnConns {
		vpnConnInfo := publishVpnConnection(vpnInfo, vpnConn)
		if vpnConnInfo != nil {
			vpnInfo.Conn[connIdx] = vpnConnInfo
			connIdx++
		}
	}

	info.InfoContent = new(zmet.ZInfoNetworkInstance_Vinfo)
	if x, ok := info.GetInfoContent().(*zmet.ZInfoNetworkInstance_Vinfo); ok {
		x.Vinfo = vpnInfo
	}

	// prepare the final stuff
	infoMsg.InfoContent = new(zmet.ZInfoMsg_Niinfo)
	if x, ok := infoMsg.GetInfoContent().(*zmet.ZInfoMsg_Niinfo); ok {
		x.Niinfo = info
	}
	publishInfo(ctx, uuid, infoMsg)
}

func handleNetworkInstanceMetricsModify(ctxArg interface{}, key string,
	statusArg interface{}) {

	log.Debugf("handleNetworkInstanceMetricsModify(%s)\n", key)
	metrics := cast.CastNetworkInstanceMetrics(statusArg)
	if metrics.Key() != key {
		log.Errorf("handleNetworkInstanceMetricsModify key/UUID mismatch %s vs %s; ignored %+v\n",
			key, metrics.Key(), metrics)
		return
	}
	log.Debugf("handleNetworkInstanceMetricsModify(%s) done\n", key)
}

func handleNetworkInstanceMetricsDelete(ctxArg interface{}, key string,
	statusArg interface{}) {

	log.Infof("handleNetworkInstanceMetricsDelete(%s)\n", key)
	metrics := cast.CastNetworkInstanceMetrics(statusArg)
	if metrics.Key() != key {
		log.Errorf("handleNetworkInstanceMetricsDelete key/UUID mismatch %s vs %s; ignored %+v\n",
			key, metrics.Key(), metrics)
		return
	}
	log.Infof("handleNetworkInstanceMetricsDelete(%s) done\n", key)
}

func createNetworkInstanceMetrics(ctx *zedagentContext, reportMetrics *zmet.ZMetricMsg) {

	sub := ctx.subNetworkInstanceMetrics
	metlist := sub.GetAll()
	if metlist == nil || len(metlist) == 0 {
		return
	}
	for _, met := range metlist {
		metrics := cast.CastNetworkInstanceMetrics(met)
		metricInstance := protoEncodeNetworkInstanceMetricProto(metrics)
		reportMetrics.Nm = append(reportMetrics.Nm, metricInstance)
	}
	log.Debugln("network instance metrics: ", reportMetrics.Nm)
}

func protoEncodeNetworkInstanceMetricProto(status types.NetworkInstanceMetrics) *zmet.ZMetricNetworkInstance {

	metric := new(zmet.ZMetricNetworkInstance)
	metric.NetworkID = status.Key()
	metric.NetworkVersion = status.UUIDandVersion.Version
	metric.Displayname = status.DisplayName
	metric.InstType = uint32(status.Type)
	switch status.Type {
	case types.NetworkInstanceTypeCloud:
		protoEncodeVpnInstanceMetric(status, metric)

	case types.NetworkInstanceTypeMesh: // XXX any subtype?
		log.Debugf("Publish Lisp Instance Metric to Zedcloud %v\n",
			metric)
		protoEncodeLispInstanceMetric(status, metric)
	}

	return metric
}

func protoEncodeLispInstanceMetric(status types.NetworkInstanceMetrics,
	metric *zmet.ZMetricNetworkInstance) {
	if status.LispMetrics == nil {
		return
	}
	metrics := status.LispMetrics

	lispGlobalMetric := new(zmet.ZMetricLispGlobal)
	lispGlobalMetric.ItrPacketSendError = &zmet.PktStat{
		Packets: metrics.ItrPacketSendError.Pkts,
		Bytes:   metrics.ItrPacketSendError.Bytes,
	}
	lispGlobalMetric.InvalidEidError = &zmet.PktStat{
		Packets: metrics.InvalidEidError.Pkts,
		Bytes:   metrics.InvalidEidError.Bytes,
	}
	lispGlobalMetric.NoDecryptKey = &zmet.PktStat{
		Packets: metrics.NoDecryptKey.Pkts,
		Bytes:   metrics.NoDecryptKey.Bytes,
	}
	lispGlobalMetric.OuterHeaderError = &zmet.PktStat{
		Packets: metrics.OuterHeaderError.Pkts,
		Bytes:   metrics.OuterHeaderError.Bytes,
	}
	lispGlobalMetric.BadInnerVersion = &zmet.PktStat{
		Packets: metrics.BadInnerVersion.Pkts,
		Bytes:   metrics.BadInnerVersion.Bytes,
	}
	lispGlobalMetric.GoodPackets = &zmet.PktStat{
		Packets: metrics.GoodPackets.Pkts,
		Bytes:   metrics.GoodPackets.Bytes,
	}
	lispGlobalMetric.ICVError = &zmet.PktStat{
		Packets: metrics.ICVError.Pkts,
		Bytes:   metrics.ICVError.Bytes,
	}
	lispGlobalMetric.LispHeaderError = &zmet.PktStat{
		Packets: metrics.LispHeaderError.Pkts,
		Bytes:   metrics.LispHeaderError.Bytes,
	}
	lispGlobalMetric.CheckSumError = &zmet.PktStat{
		Packets: metrics.CheckSumError.Pkts,
		Bytes:   metrics.CheckSumError.Bytes,
	}
	lispGlobalMetric.DecapReInjectError = &zmet.PktStat{
		Packets: metrics.DecapReInjectError.Pkts,
		Bytes:   metrics.DecapReInjectError.Bytes,
	}
	lispGlobalMetric.DecryptError = &zmet.PktStat{
		Packets: metrics.DecryptError.Pkts,
		Bytes:   metrics.DecryptError.Bytes,
	}
	metric.LispGlobalStats = lispGlobalMetric

	flowStats := []*zmet.ZMetricFlow{}

	for _, eidStat := range metrics.EidStats {
		iid := eidStat.IID
		metricFlow := &zmet.ZMetricFlow{
			Iid: iid,
		}
		lEndPoint := &zmet.ZMetricFlowEndPoint{}
		flowLinks := []*zmet.ZMetricFlowLink{}
		for _, eidMap := range metrics.EidMaps {
			for _, eid := range eidMap.Eids {
				flowLink := &zmet.ZMetricFlowLink{}
				flowLink.Link = new(zmet.ZMetricFlowLink_Eid)
				if x, ok := flowLink.GetLink().(*zmet.ZMetricFlowLink_Eid); ok {
					x.Eid = eid.String()
				}
				flowLinks = append(flowLinks, flowLink)
			}
		}
		lEndPoint.Link = flowLinks
		metricFlow.LEndPoint = lEndPoint

		rEndPoint := []*zmet.ZMetricFlowEndPoint{}
		eid := eidStat.Eid
		for _, rlocStat := range eidStat.RlocStats {
			rloc := rlocStat.Rloc
			stat := rlocStat.Stats
			flowEndPoint := &zmet.ZMetricFlowEndPoint{}
			flowEndPoint.Stats = &zmet.PktStat{
				Packets: stat.Pkts,
				Bytes:   stat.Bytes,
			}
			flowEndPoint.Endpoint = new(zmet.ZMetricFlowEndPoint_Rloc)
			if x, ok := flowEndPoint.GetEndpoint().(*zmet.ZMetricFlowEndPoint_Rloc); ok {
				x.Rloc = rloc.String()
			}
			flowLinks := []*zmet.ZMetricFlowLink{}
			flowLink := &zmet.ZMetricFlowLink{}
			flowLink.Link = new(zmet.ZMetricFlowLink_Eid)
			if x, ok := flowLink.GetLink().(*zmet.ZMetricFlowLink_Eid); ok {
				x.Eid = eid.String()
			}
			flowLinks = append(flowLinks, flowLink)
			flowEndPoint.Link = flowLinks

			rEndPoint = append(rEndPoint, flowEndPoint)
		}
		metricFlow.REndPoint = rEndPoint
		flowStats = append(flowStats, metricFlow)
	}
	metric.FlowStats = flowStats

	// Fill lisp metric stats also for now.
	// We can deprecate the same later
	lispMetric := new(zmet.ZMetricLisp)
	lispMetric.ItrPacketSendError = &zmet.PktStat{
		Packets: metrics.ItrPacketSendError.Pkts,
		Bytes:   metrics.ItrPacketSendError.Bytes,
	}
	lispMetric.InvalidEidError = &zmet.PktStat{
		Packets: metrics.InvalidEidError.Pkts,
		Bytes:   metrics.InvalidEidError.Bytes,
	}
	lispMetric.NoDecryptKey = &zmet.PktStat{
		Packets: metrics.NoDecryptKey.Pkts,
		Bytes:   metrics.NoDecryptKey.Bytes,
	}
	lispMetric.OuterHeaderError = &zmet.PktStat{
		Packets: metrics.OuterHeaderError.Pkts,
		Bytes:   metrics.OuterHeaderError.Bytes,
	}
	lispMetric.BadInnerVersion = &zmet.PktStat{
		Packets: metrics.BadInnerVersion.Pkts,
		Bytes:   metrics.BadInnerVersion.Bytes,
	}
	lispMetric.GoodPackets = &zmet.PktStat{
		Packets: metrics.GoodPackets.Pkts,
		Bytes:   metrics.GoodPackets.Bytes,
	}
	lispMetric.ICVError = &zmet.PktStat{
		Packets: metrics.ICVError.Pkts,
		Bytes:   metrics.ICVError.Bytes,
	}
	lispMetric.LispHeaderError = &zmet.PktStat{
		Packets: metrics.LispHeaderError.Pkts,
		Bytes:   metrics.LispHeaderError.Bytes,
	}
	lispMetric.CheckSumError = &zmet.PktStat{
		Packets: metrics.CheckSumError.Pkts,
		Bytes:   metrics.CheckSumError.Bytes,
	}
	lispMetric.DecapReInjectError = &zmet.PktStat{
		Packets: metrics.DecapReInjectError.Pkts,
		Bytes:   metrics.DecapReInjectError.Bytes,
	}
	lispMetric.DecryptError = &zmet.PktStat{
		Packets: metrics.DecryptError.Pkts,
		Bytes:   metrics.DecryptError.Bytes,
	}

	lispStats := []*zmet.EidStats{}
	for _, eidStat := range metrics.EidStats {
		lispStat := &zmet.EidStats{
			IID: eidStat.IID,
			EID: eidStat.Eid.String(),
		}

		rlocStats := []*zmet.RlocStats{}
		for _, rloc := range eidStat.RlocStats {
			rlocStat := &zmet.RlocStats{
				Rloc: rloc.Rloc.String(),
				Stats: &zmet.PktStat{
					Packets: rloc.Stats.Pkts,
					Bytes:   rloc.Stats.Bytes,
				},
				SecondsSinceLastPacket: rloc.SecondsSinceLastPacket,
			}
			rlocStats = append(rlocStats, rlocStat)
		}
		lispStat.RlocStatsEntries = rlocStats
		lispStats = append(lispStats, lispStat)
	}
	lispMetric.EidStatsEntries = lispStats

	metric.InstanceContent = new(zmet.ZMetricNetworkInstance_Lispm)
	if x, ok := metric.GetInstanceContent().(*zmet.ZMetricNetworkInstance_Lispm); ok {
		x.Lispm = lispMetric
	}
}

func protoEncodeVpnInstanceMetric(metrics types.NetworkInstanceMetrics,
	instanceMetrics *zmet.ZMetricNetworkInstance) {

	if metrics.VpnMetrics == nil {
		return
	}

	stats := metrics.VpnMetrics
	vpnMetric := new(zmet.ZMetricVpn)
	vpnMetric.ConnStat = protoEncodeVpnInstanceStat(stats.DataStat)
	vpnMetric.NatTStat = protoEncodeVpnInstanceStat(stats.NatTStat)
	vpnMetric.IkeStat = protoEncodeVpnInstanceStat(stats.IkeStat)
	vpnMetric.EspStat = protoEncodeVpnInstanceStat(stats.EspStat)

	instanceMetrics.InstanceContent = new(zmet.ZMetricNetworkInstance_Vpnm)
	if x, ok := instanceMetrics.GetInstanceContent().(*zmet.ZMetricNetworkInstance_Vpnm); ok {
		x.Vpnm = vpnMetric
	}
	protoEncodeVpnInstanceFlowMetric(metrics, instanceMetrics)
}

func protoEncodeVpnInstanceStat(stats types.LinkPktStats) *zmet.ZMetricConn {
	connStat := new(zmet.ZMetricConn)
	connStat.InPkts = new(zmet.PktStat)
	connStat.OutPkts = new(zmet.PktStat)
	connStat.InPkts.Packets = stats.InPkts.Pkts
	connStat.InPkts.Bytes = stats.InPkts.Bytes
	connStat.OutPkts.Packets = stats.OutPkts.Pkts
	connStat.OutPkts.Bytes = stats.OutPkts.Bytes
	return connStat
}

func protoEncodeVpnInstanceFlowMetric(metrics types.NetworkInstanceMetrics,
	instanceMetrics *zmet.ZMetricNetworkInstance) {

	if len(metrics.VpnMetrics.VpnConns) == 0 {
		return
	}

	vpnMetrics := metrics.VpnMetrics
	instanceMetrics.FlowStats = make([]*zmet.ZMetricFlow,
		len(vpnMetrics.VpnConns))
	for idx, connStats := range vpnMetrics.VpnConns {

		flowStats := new(zmet.ZMetricFlow)
		flowStats.Id = connStats.Id
		flowStats.Name = connStats.Name
		flowStats.Type = uint32(connStats.Type)
		flowStats.EstTime = connStats.EstTime

		lEndPoint := protoEncodeVpnMetricEndPtIpAddr(connStats.LEndPoint)
		lEndPoint.Stats = protoEncodeVpnMetricStats(connStats.LEndPoint.PktStats)
		lLink := protoEncodeVpnMetricLink(connStats.LEndPoint.LinkInfo)
		lEndPoint.Link = make([]*zmet.ZMetricFlowLink, 1)
		lEndPoint.Link[0] = lLink

		rEndPoint := protoEncodeVpnMetricEndPtIpAddr(connStats.REndPoint)
		rEndPoint.Stats = protoEncodeVpnMetricStats(connStats.REndPoint.PktStats)
		rLink := protoEncodeVpnMetricLink(connStats.REndPoint.LinkInfo)
		rEndPoint.Link = make([]*zmet.ZMetricFlowLink, 1)
		rEndPoint.Link[0] = rLink

		flowStats.LEndPoint = lEndPoint
		flowStats.REndPoint = make([]*zmet.ZMetricFlowEndPoint, 1)
		flowStats.REndPoint[0] = rEndPoint
		instanceMetrics.FlowStats[idx] = flowStats
	}
}

func protoEncodeVpnMetricEndPtIpAddr(endPInfo types.VpnEndPointMetrics) *zmet.ZMetricFlowEndPoint {
	endPoint := new(zmet.ZMetricFlowEndPoint)
	endPoint.Endpoint = new(zmet.ZMetricFlowEndPoint_IpAddr)
	if x, ok := endPoint.GetEndpoint().(*zmet.ZMetricFlowEndPoint_IpAddr); ok {
		x.IpAddr = endPInfo.IpAddr
	}
	return endPoint
}

func protoEncodeVpnMetricLink(linkInfo types.VpnLinkMetrics) *zmet.ZMetricFlowLink {
	link := new(zmet.ZMetricFlowLink)
	link.SpiId = linkInfo.SpiId
	link.Link = new(zmet.ZMetricFlowLink_SubNet)
	if x, ok := link.GetLink().(*zmet.ZMetricFlowLink_SubNet); ok {
		x.SubNet = linkInfo.SubNet
	}
	return link
}

func protoEncodeVpnMetricStats(linkStats types.PktStats) *zmet.PktStat {
	pktStats := new(zmet.PktStat)
	pktStats.Bytes = linkStats.Bytes
	pktStats.Packets = linkStats.Pkts
	return pktStats
}
