package api

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/qlcchain/go-qlc/common/types"
	"github.com/qlcchain/go-qlc/consensus"
	"github.com/qlcchain/go-qlc/ledger"
	"github.com/qlcchain/go-qlc/log"
	"github.com/qlcchain/go-qlc/p2p"
	"github.com/qlcchain/go-qlc/p2p/protos"
	"github.com/qlcchain/go-qlc/test/mock"
	"go.uber.org/zap"
)

type QlcApi struct {
	ledger *ledger.Ledger
	dpos   *consensus.DposService
	logger *zap.SugaredLogger
}

type TokenPending struct {
	PendingInfo *types.PendingInfo `json:"pendingInfo"`
	TokenName   string             `json:"tokenName"`
	Hash        types.Hash         `json:"hash"`
}

func NewQlcApi(l *ledger.Ledger, dpos *consensus.DposService) *QlcApi {
	return &QlcApi{ledger: l, dpos: dpos, logger: log.NewLogger("rpcapi")}
}

func (q *QlcApi) AccountsBalances(addresses []types.Address) (map[types.Address]map[types.Hash][]types.Balance, error) {
	q.logger.Info("addresses", addresses)
	r := make(map[types.Address]map[types.Hash][]types.Balance)
	for _, addr := range addresses {
		ac, err := q.ledger.GetAccountMeta(addr)
		if err != nil {
			return nil, err
		}
		t := make(map[types.Hash][]types.Balance)
		for _, token := range ac.Tokens {
			t[token.Type] = append(t[token.Type], token.Balance)
			pendings, err := q.ledger.TokenPendingInfo(addr, token.Type)
			if err != nil {
				return nil, err
			}
			b := types.ZeroBalance
			for _, pending := range pendings {
				b = b.Add(pending.Amount)
			}
			t[token.Type] = append(t[token.Type], b)
		}
		r[addr] = t
	}
	return r, nil
}

func (q *QlcApi) AccountsFrontiers(addresses []types.Address) (map[types.Address]map[types.Hash]types.Hash, error) {
	q.logger.Info("addresses", addresses)
	r := make(map[types.Address]map[types.Hash]types.Hash)
	for _, addr := range addresses {
		ac, err := q.ledger.GetAccountMeta(addr)
		if err != nil {
			return nil, err
		}
		t := make(map[types.Hash]types.Hash)
		for _, token := range ac.Tokens {
			t[token.Type] = token.Header
		}
		r[addr] = t
	}
	return r, nil
}

func (q *QlcApi) AccountsPending(addresses []types.Address, n int) (map[types.Address][]*TokenPending, error) {
	q.logger.Info("addresses", addresses)
	apMap := make(map[types.Address][]*TokenPending)
	for _, addr := range addresses {
		pendingkeys, err := q.ledger.Pending(addr)
		if err != nil {
			return nil, err
		}
		tps := make([]*TokenPending, 0)

		for _, pendingkey := range pendingkeys {
			if len(tps) >= n {
				break
			}
			pendinginfo, err := q.ledger.GetPending(*pendingkey)
			if err != nil {
				return nil, err
			}

			token, err := mock.GetTokenById(pendinginfo.Type)
			if err != nil {
				return nil, err
			}
			tokenname := token.TokenName
			tp := TokenPending{
				PendingInfo: pendinginfo,
				TokenName:   tokenname,
				Hash:        pendingkey.Hash,
			}
			tps = append(tps, &tp)
		}
		apMap[addr] = tps
	}
	return apMap, nil
}

func (q *QlcApi) GetOnlineRepresentatives() []types.Address {
	as := q.dpos.GetOnlineRepresentatives()
	if as == nil {
		return make([]types.Address, 0)
	}
	return as
}

func (q *QlcApi) BlocksInfo(hash []types.Hash) ([]*APIBlock, error) {
	bs := make([]*APIBlock, 0)
	for _, h := range hash {
		b := new(APIBlock)
		q.logger.Debug(h.String())
		block, err := q.ledger.GetStateBlock(h)
		q.logger.Debug(block)
		if err != nil {
			return nil, fmt.Errorf("%s, %s", h, err)
		}
		b = b.fromStateBlock(block)
		b.SubType, b.Amount, err = q.judgeBlockKind(block)
		if err != nil {
			return nil, fmt.Errorf("%s, %s", h, err)
		}
		//b.SubType = "state"
		q.logger.Info("getToken,", block.GetToken())
		token, err := mock.GetTokenById(block.GetToken())
		if err != nil {
			return nil, fmt.Errorf("%s, %s", h, err)
		}
		b.TokenName = token.TokenName
		bs = append(bs, b)
	}
	return bs, nil
}

func (q *QlcApi) Process(block *types.StateBlock) (types.Hash, error) {
	flag, err := q.ledger.Process(block)
	if err != nil {
		return types.ZeroHash, err
	}

	//if flag != ledger.Other {
	//
	//	return block.GetHash(), nil
	//} else {
	//	return types.ZeroHash, fmt.Errorf("%d", flag)
	//}

	q.logger.Info("process result, ", flag)
	switch flag {
	case ledger.Progress:
		pushBlock := protos.PublishBlock{
			Blk: block,
		}
		bytes, err := protos.PublishBlockToProto(&pushBlock)
		if err != nil {
			q.logger.Error(err)
			return types.ZeroHash, err
		} else {
			q.logger.Info("broadcast block")
			q.dpos.GetP2PService().Broadcast(p2p.PublishReq, bytes)
			return block.GetHash(), nil
		}
	case ledger.BadWork:
		return types.ZeroHash, errors.New("bad work")
	case ledger.BadSignature:
		return types.ZeroHash, errors.New("bad signature")
	case ledger.Old:
		return types.ZeroHash, errors.New("old block")
	case ledger.Fork:
		return types.ZeroHash, errors.New("fork")
	case ledger.GapSource:
		return types.ZeroHash, errors.New("gap source block")
	case ledger.GapPrevious:
		return types.ZeroHash, errors.New("gap previous block")
	case ledger.BalanceMismatch:
		return types.ZeroHash, errors.New("balance mismatch")
	case ledger.UnReceivable:
		return types.ZeroHash, errors.New("unReceivable")
	default:
		return types.ZeroHash, errors.New("error processing block")
	}
}

func (q *QlcApi) judgeBlockKind(block *types.StateBlock) (string, types.Balance, error) {
	hash := block.GetHash()
	q.logger.Debug(hash.String())
	blkType, err := q.ledger.JudgeBlockKind(hash)
	if err != nil {
		return "", types.ZeroBalance, err
	}
	prevBlock, _ := q.ledger.GetStateBlock(block.Previous)
	switch blkType {
	case ledger.Open:
		return "open", block.Balance, nil
	case ledger.Receive:
		return "receive", block.Balance.Sub(prevBlock.Balance), nil
	case ledger.Send:
		return "send", prevBlock.Balance.Sub(block.Balance), nil
	case ledger.Change:
		return "change", types.ZeroBalance, nil
	default:
		return "unknow", types.ZeroBalance, nil
	}
}

func (q *QlcApi) AccountHistoryTopn(address types.Address, n int) ([]*APIBlock, error) {
	q.logger.Info(address)
	bs := make([]*APIBlock, 0)
	ac, err := q.ledger.GetAccountMeta(address)
	if err != nil {
		return nil, err
	}
	fmt.Println("ac", ac)
	for _, token := range ac.Tokens {
		h := token.Header
		count_limit := 0

		for count_limit < n {

			block, err := q.ledger.GetStateBlock(h)

			if err != nil {
				if err == ledger.ErrBlockNotFound {
					break
				}
				return nil, err
			}

			b := new(APIBlock)
			b.SubType, b.Amount, err = q.judgeBlockKind(block)
			if err != nil {
				q.logger.Info(err)
				return nil, err
			}
			q.logger.Info("token,", block.GetToken())
			token, err := mock.GetTokenById(block.GetToken())
			if err != nil {
				q.logger.Info(err)
				return nil, err
			}
			b.TokenName = token.TokenName
			b = b.fromStateBlock(block)
			bs = append(bs, b)

			h = block.GetPrevious()
			count_limit = count_limit + 1
		}
	}
	return bs, nil
}

func (q *QlcApi) AccountInfo(addr types.Address) (*types.AccountMeta, error) {
	am, err := q.ledger.GetAccountMeta(addr)
	if err != nil {
		return nil, err
	}
	return am, nil
}

func (q *QlcApi) ValidateAccount(addr string) bool {
	return types.IsValidHexAddress(addr)
}

func (q *QlcApi) Tokens() ([]*mock.TokenInfo, error) {
	var tis []*mock.TokenInfo
	scs := mock.GetSmartContracts()
	for _, sc := range scs {
		hash := sc.GetHash()
		ti, err := mock.GetTokenById(hash)
		if err != nil {
			return nil, err
		}
		tis = append(tis, &ti)
	}
	return tis, nil
}
