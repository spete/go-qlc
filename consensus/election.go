package consensus

import (
	"github.com/qlcchain/go-qlc/common/types"
	"github.com/qlcchain/go-qlc/p2p"
	"github.com/qlcchain/go-qlc/p2p/protos"
)

type electionStatus struct {
	winner types.Block
	tally  types.Balance
}

type Election struct {
	vote          *Votes
	status        electionStatus
	confirmed     bool
	dps           *DposService
	announcements uint
	supply        types.Balance
}

func NewElection(dps *DposService, block types.Block) (*Election, error) {
	vt := NewVotes(block)
	status := electionStatus{block, types.ZeroBalance}
	b1 := types.StringToBalance("30000000000000000")

	return &Election{
		vote:          vt,
		status:        status,
		confirmed:     false,
		supply:        b1,
		dps:           dps,
		announcements: 0,
	}, nil
}

func (el *Election) voteAction(va *protos.ConfirmAckBlock) {
	valid := IsAckSignValidate(va)
	if valid != true {
		return
	}
	var shouldProcess bool
	exit, vt := el.vote.voteExit(va.Account)
	if exit == true {
		if vt.Sequence < va.Sequence {
			shouldProcess = true
		} else {
			shouldProcess = false
		}
	} else {
		shouldProcess = true
	}
	if shouldProcess {
		el.vote.repVotes[va.Account] = va
		data, err := protos.ConfirmAckBlockToProto(va)
		if err != nil {
			el.dps.logger.Error("vote to proto error")
		}
		el.dps.ns.Broadcast(p2p.ConfirmAck, data)
	}
	el.haveQuorum()
}

func (el *Election) haveQuorum() {
	t := el.tally()
	if !(len(t) > 0) {
		return
	}
	var hash types.Hash
	var balance = types.ZeroBalance
	for key, value := range t {
		if balance.Compare(value) == types.BalanceCompSmaller {
			balance = value
			hash = key
		}
	}
	blk, err := el.dps.ledger.GetStateBlock(hash)
	if err != nil {
		el.dps.logger.Infof("err:[%s] when get block", err)
	}
	if balance.Compare(el.supply) == types.BalanceCompBigger {
		el.dps.logger.Infof("hash:%s block has confirmed", blk.GetHash())
		el.status.winner = blk
		el.confirmed = true
		el.status.tally = balance
	} else {
		el.dps.logger.Infof("wait for enough rep vote,current vote is [%s]", balance.String())
	}
}

func (el *Election) tally() map[types.Hash]types.Balance {
	totals := make(map[types.Hash]types.Balance)
	for key, value := range el.vote.repVotes {
		if _, ok := totals[value.Blk.GetHash()]; !ok {
			totals[value.Blk.GetHash()] = types.ZeroBalance
		}
		weight := el.dps.ledger.Weight(key)
		totals[value.Blk.GetHash()] = totals[value.Blk.GetHash()].Add(weight)
	}
	return totals
}
