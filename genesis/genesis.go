package genesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/util"
	"github.com/zarbchain/zarb-go/validator"
)

// How many bytes to take from the front of the Genesis hash to append
// to the ChainName to form the ChainID. The idea is to avoid some classes
// of replay attack between chains with the same name.
const shortHashSuffixBytes = 3

// core types for a genesis definition

type genAccount struct {
	Address crypto.Address
	Balance int64
}

type genValidator struct {
	Address   crypto.Address
	Stake     int64
	PublicKey crypto.PublicKey
}

// Genesis is stored in the state database
type Genesis struct {
	data genesisData
}

type genesisData struct {
	ChainName   string
	GenesisTime time.Time
	Accounts    []genAccount
	Validators  []genValidator
}

func (gen *Genesis) Hash() crypto.Hash {
	bs, err := gen.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("could not create hash of Genesis: %v", err))
	}
	return crypto.HashH(bs)
}

func (gen *Genesis) ChainName() string {
	return gen.data.ChainName
}

func (gen *Genesis) IsForMainnet() bool {
	return gen.data.ChainName == "zarb-mainnet"
}

func (gen *Genesis) IsForTestnet() bool {
	return gen.data.ChainName == "zarb-testnet"
}

func (gen *Genesis) IsForTest() bool {
	return !(gen.IsForMainnet() || gen.IsForTestnet())
}

func (gen *Genesis) GenesisTime() time.Time {
	return gen.data.GenesisTime
}

func (gen *Genesis) Accounts() []*account.Account {
	accs := make([]*account.Account, 0)
	for _, genAcc := range gen.data.Accounts {
		acc := account.NewAccount(genAcc.Address)
		acc.AddToBalance(genAcc.Balance)
		accs = append(accs, acc)
	}

	return accs
}

func (gen *Genesis) Validators() []*validator.Validator {
	vals := make([]*validator.Validator, 0, len(gen.data.Validators))
	for _, genVal := range gen.data.Validators {
		val := validator.NewValidator(genVal.PublicKey, 0)
		val.AddToStake(genVal.Stake)
		vals = append(vals, val)
	}

	return vals
}

func (gen *Genesis) ValidatorsAddress() []crypto.Address {
	var vals []crypto.Address
	for _, genVal := range gen.data.Validators {
		vals = append(vals, genVal.Address)
	}

	return vals
}

func (gen Genesis) MarshalJSON() ([]byte, error) {
	return json.Marshal(&gen.data)
}

func (gen *Genesis) UnmarshalJSON(bs []byte) error {
	return json.Unmarshal(bs, &gen.data)
}

func makeGenesisAccount(acc *account.Account) genAccount {
	return genAccount{
		Address: acc.Address(),
		Balance: acc.Balance(),
	}
}

func makeGenesisValidator(val *validator.Validator) genValidator {
	return genValidator{
		PublicKey: val.PublicKey(),
		Address:   val.Address(),
		Stake:     val.Stake(),
	}
}

func MakeGenesis(chainName string, genesisTime time.Time,
	accounts []*account.Account,
	validators []*validator.Validator) *Genesis {

	genAccs := make([]genAccount, 0, len(accounts))
	for _, acc := range accounts {
		genAcc := makeGenesisAccount(acc)
		genAccs = append(genAccs, genAcc)
	}

	genVals := make([]genValidator, 0, len(validators))
	for _, val := range validators {
		genVal := makeGenesisValidator(val)
		genVals = append(genVals, genVal)
	}

	return &Genesis{
		data: genesisData{
			ChainName:   chainName,
			GenesisTime: genesisTime,
			Accounts:    genAccs,
			Validators:  genVals,
		},
	}
}

// LoadFromFile loads genesis object from a JSON file
func LoadFromFile(file string) (*Genesis, error) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var gen Genesis
	if err := json.Unmarshal(dat, &gen); err != nil {
		return nil, err
	}
	return &gen, nil
}

// SaveToFile saves the genesis info a JSON file
func (gen *Genesis) SaveToFile(file string) error {
	json, err := gen.MarshalJSON()
	if err != nil {
		return err
	}

	// write  dataContent to file
	if err := util.WriteFile(file, json); err != nil {
		return err
	}

	return nil
}
