package packagemanager

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/iden3/contracts-abi/state/go/abi"
	"github.com/iden3/go-circuits/v2"
	core "github.com/iden3/go-iden3-core/v2"
	"github.com/iden3/go-jwz/v2"
	"github.com/iden3/iden3comm/v2"
	"github.com/iden3/iden3comm/v2/packers"
	"github.com/pkg/errors"
)

type state struct {
	contracts                map[int]*abi.State
	globalStateValidDuration time.Duration
}

func registerCustomDIDMethods(cdm []CustomDIDMethods) error {
	for _, network := range cdm {
		params := core.DIDMethodNetworkParams{
			Method:      core.DIDMethodPolygonID,
			Blockchain:  core.Blockchain(network.Blockchain),
			Network:     core.NetworkID(network.Network),
			NetworkFlag: network.NetworkFlag,
		}
		err := core.RegisterDIDMethodNetwork(params, core.WithChainID(network.ChainID))
		if err != nil {
			return errors.Errorf("did method can't be registered: %v", err)
		}
	}
	return nil
}

func (s *state) verify(_ circuits.CircuitID, pubsignals []string) error {
	bytePubsig, err := json.Marshal(pubsignals)
	if err != nil {
		return errors.Errorf("error marshaling pubsignals: %v", err)
	}

	authPubSignals := circuits.AuthV2PubSignals{}
	err = authPubSignals.PubSignalsUnmarshal(bytePubsig)
	if err != nil {
		return errors.Errorf("error unmarshaling pubsignals: %v", err)
	}

	userDID, err := core.ParseDIDFromID(*authPubSignals.UserID)
	if err != nil {
		return errors.Errorf("error converting userID '%s' to userDID: %v",
			authPubSignals.UserID.String(), err)
	}

	chainID, err := core.ChainIDfromDID(*userDID)
	if err != nil {
		return errors.Errorf("error extracting chainID from userDID '%s': %v",
			userDID.String(), err)
	}

	contract, ok := s.contracts[int(chainID)]
	if !ok {
		return errors.Errorf("not supported chainID '%d'", chainID)
	}

	globalState := authPubSignals.GISTRoot.BigInt()
	globalStateInfo, err := contract.GetGISTRootInfo(&bind.CallOpts{}, globalState)
	if err != nil {
		return errors.Errorf("error getting global state info by state '%s': %v",
			globalState, err)
	}
	if (big.NewInt(0)).Cmp(globalStateInfo.CreatedAtTimestamp) == 0 {
		return errors.Errorf("root %s doesn't exist in smart contract",
			globalState.String())
	}
	if globalState.Cmp(globalStateInfo.Root) != 0 {
		return errors.Errorf("invalid global state info in the smart contract, expected root %s, got %s",
			globalState.String(), globalStateInfo.Root.String())
	}

	if (big.NewInt(0)).Cmp(globalStateInfo.ReplacedByRoot) != 0 &&
		time.Since(time.Unix(globalStateInfo.ReplacedAtTimestamp.Int64(), 0)) > s.globalStateValidDuration {
		return errors.Errorf("global state is too old, replaced timestamp is %v",
			globalStateInfo.ReplacedAtTimestamp.Int64())
	}

	return nil
}

type Options struct {
	VerificationKeyPath      string
	GlobalStateValidDuration time.Duration
	CustomDIDMethods         []CustomDIDMethods `mapstructure:"-"`
}

type Option func(*Options)

// CustomDIDMethods struct
// Example: SUPPORTED_CUSTOM_DID_METHODS='[{"blockchain":"linea","network":"testnet","networkFlag":"0b01000001","chainID":59140}]'
type CustomDIDMethods struct {
	Blockchain  string `tip:"Identity blockchain for custom network"`
	Network     string `tip:"Identity network for custom network"`
	NetworkFlag byte   `tip:"Identity network flag for custom network"`
	ChainID     int    `tip:"Chain id for custom network"`
}

// UnmarshalJSON implements the Unmarshal interface for CustomDIDMethods
func (cn *CustomDIDMethods) UnmarshalJSON(data []byte) error {
	aux := struct {
		Blockchain  string `json:"blockchain"`
		Network     string `json:"network"`
		NetworkFlag string `json:"networkFlag"`
		ChainID     int    `json:"chainId"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.NetworkFlag) != 10 || aux.NetworkFlag[:2] != "0b" {
		return errors.New("invalid NetworkFlag format")
	}
	flag, err := strconv.ParseUint(aux.NetworkFlag[2:], 2, 8)
	if err != nil {
		return err
	}

	cn.Blockchain = aux.Blockchain
	cn.Network = aux.Network
	cn.NetworkFlag = byte(flag)
	cn.ChainID = aux.ChainID

	return nil
}

func WithVerificationKeyPath(path string) Option {
	return func(o *Options) {
		o.VerificationKeyPath = path
	}
}

func WithGlobalStateValidDuration(duration time.Duration) Option {
	return func(o *Options) {
		o.GlobalStateValidDuration = duration
	}
}

func WithCustomDIDMethods(jsonStr string) Option {
	return func(o *Options) {
		var customDIDMethods []CustomDIDMethods
		if jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &customDIDMethods); err != nil {
				customDIDMethods = nil
			}
		}
		o.CustomDIDMethods = customDIDMethods
	}
}

func NewPackageManager(
	supportedRPC map[string]string,
	supportedStateContracts map[string]string,
	opts ...Option,
) (*iden3comm.PackageManager, error) {

	options := &Options{
		VerificationKeyPath:      "/keys",
		GlobalStateValidDuration: time.Minute * 15,
		CustomDIDMethods:         []CustomDIDMethods{},
	}
	for _, opt := range opts {
		opt(options)
	}

	err := registerCustomDIDMethods(options.CustomDIDMethods)
	if err != nil {
		return nil, err
	}

	authV2VerificationKeyPath := fmt.Sprintf("%s/authV2.json", options.VerificationKeyPath)
	verificationKey, err := os.ReadFile(authV2VerificationKeyPath)
	if err != nil {
		return nil, errors.Errorf(
			"issuer with the file verification_key.json by path '%s': %v", authV2VerificationKeyPath, err)
	}

	states := state{
		contracts:                make(map[int]*abi.State, len(supportedStateContracts)),
		globalStateValidDuration: options.GlobalStateValidDuration,
	}
	for chainID, stateAddr := range supportedStateContracts {
		rpcURL, ok := supportedRPC[chainID]
		if !ok {
			return nil, errors.Errorf("not supported RPC for blockchain %s", chainID)
		}
		ec, err := ethclient.Dial(rpcURL)
		if err != nil {
			return nil, err
		}
		stateContract, err := abi.NewState(common.HexToAddress(stateAddr), ec)
		if err != nil {
			return nil, err
		}
		v, err := strconv.Atoi(chainID)
		if err != nil {
			return nil, errors.Errorf("invalid chainID '%s': %v", chainID, err)
		}
		states.contracts[v] = stateContract
	}

	verifications := make(map[jwz.ProvingMethodAlg]packers.VerificationParams)
	verifications[jwz.AuthV2Groth16Alg] = packers.NewVerificationParams(
		verificationKey,
		states.verify,
	)

	zkpPackerV2 := packers.NewZKPPacker(
		nil,
		verifications,
	)

	packageManager := iden3comm.NewPackageManager()

	err = packageManager.RegisterPackers(zkpPackerV2, &packers.PlainMessagePacker{})
	if err != nil {
		return nil, err
	}

	return packageManager, nil
}
