// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"

	"storj.io/storj/pkg/dht"
	"storj.io/storj/pkg/node"
	"storj.io/storj/pkg/pb"
	"storj.io/storj/pkg/pointerdb"
	"storj.io/storj/pkg/utils"
	"storj.io/storj/storage"
)

// Collector is the interface for the data accounting collector
type Collector interface {
	Run(ctx context.Context) error
}

type collector struct {
	pointerdb *pointerdb.Server
	overlay   pb.OverlayServer
	limit     int
	logger    *zap.Logger
	ticker    *time.Ticker
}

// NewCollector creates a new instance of collector
func NewCollector(pointerdb *pointerdb.Server, overlay pb.OverlayServer, limit int, logger *zap.Logger, interval time.Duration) *collector {
	return &collector{
		pointerdb: pointerdb,
		overlay:   overlay,
		limit:     limit,
		logger:    logger,
	}
}

// Run the collector loop
func (c *collector) Run(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)

	for {
		err = c.identifyActiveNodes(ctx)
		if err != nil {
			zap.L().Error("Collector failed", zap.Error(err))
		}

		select {
		case <-c.ticker.C: // wait for the next interval to happen
		case <-ctx.Done(): // or the collector is canceled via context
			return ctx.Err()
		}
	}
}

//identifyActiveNodes iterates through pointerdb and identifies nodes that have storage on them
func (c *collector) identifyActiveNodes(ctx context.Context) (err error) {
	defer mon.Task()(&ctx)(&err)
	c.logger.Debug("entering pointerdb iterate")

	err = c.pointerdb.Iterate(ctx, &pb.IterateRequest{Recurse: true},
		func(it storage.Iterator) error {
			var item storage.ListItem
			lim := c.limit
			if lim <= 0 || lim > storage.LookupLimit {
				lim = storage.LookupLimit
			}
			for ; lim > 0 && it.Next(&item); lim-- {
				pointer := &pb.Pointer{}
				err = proto.Unmarshal(item.Value, pointer)
				if err != nil {
					return Error.New("error unmarshalling pointer %s", err)
				}
				pieces := pointer.Remote.RemotePieces
				var nodeIDs []dht.NodeID
				for _, p := range pieces {
					nodeIDs = append(nodeIDs, node.IDFromString(p.NodeId))
				}
				online, err := c.onlineNodes(ctx, nodeIDs)
				if err != nil {
					return Error.New("error getting online nodes %s", err)
				}
				go c.tallyAtRestStorage(pointer, online)
			}
			return nil
		},
	)
	return err
}

func (c *collector) onlineNodes(ctx context.Context, nodeIDs []dht.NodeID) (online []*pb.Node, err error) {
	responses, err := c.overlay.BulkLookup(ctx, utils.NodeIDsToLookupRequests(nodeIDs))
	if err != nil {
		return []*pb.Node{}, err
	}
	nodes := utils.LookupResponsesToNodes(responses)
	for _, n := range nodes {
		if n != nil {
			online = append(online, n)
		}
	}
	return online, nil
}

func (c *collector) tallyAtRestStorage(pointer *pb.Pointer, nodes []*pb.Node) {
	//Iterate through identified nodes.
	//if we have not contacted the node previously this process,
	// validate they are accessible on the network
	//function
	//If reachable, update the granular data table to include
	//the amount stored for this piece.
	segmentSize := pointer.GetSize()
	minReq := pointer.Remote.Redundancy.GetMinReq()
	if minReq <= 0 {
		zap.L().Error("minReq must be an int greater than 0")
		return
	}
	pieceSize := segmentSize / int64(minReq)
	for _, n := range nodes {
		fmt.Print(n) //placeholder
		fmt.Print(pieceSize) //placeholder
		// lr, err := c.overlay.Lookup(ctx, &pb.LookupRequest{n.Id})?
		//ping n ?
		go c.updateGranularTable(n.Id, pieceSize)
	}
}

func (c *collector) updateGranularTable(nodeID string, pieceSize int64) {
	//TODO
}

func (c *collector) tallyBandwith() {
	//TODO
	//Query bandwidth allocation database, selecting all new
	//contracts since the last collection run time. grouping by storage
	//node ID and adding total of bandwidth to granular data table.
}
