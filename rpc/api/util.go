package api

import (
	"github.com/qlcchain/go-qlc/common/types"
	"github.com/qlcchain/go-qlc/common/util"
	"github.com/qlcchain/go-qlc/log"
	"github.com/qlcchain/go-qlc/test/mock"
	"go.uber.org/zap"
)

type UtilApi struct {
	logger *zap.SugaredLogger
}

func NewUtilApi() *UtilApi {
	return &UtilApi{logger: log.NewLogger("util_account")}
}

func (u *UtilApi) Decrypt(cryptograph string, passphrase string) (string, error) {
	return util.Decrypt(cryptograph, passphrase)
}

func (u *UtilApi) Encrypt(raw string, passphrase string) (string, error) {
	return util.Encrypt(raw, passphrase)
}

func (u *UtilApi) RawToBalance(balance types.Balance, unit string, tokenName *string) (types.Balance, error) {
	b, err := mock.RawToBalance(balance, unit)
	if err != nil {
		return types.ZeroBalance, err
	}
	return b, nil
}

func (u *UtilApi) BalanceToRaw(balance types.Balance, unit string, tokenName *string) (types.Balance, error) {
	b, err := mock.BalanceToRaw(balance, unit)
	if err != nil {
		return types.ZeroBalance, err
	}
	return b, nil
}
