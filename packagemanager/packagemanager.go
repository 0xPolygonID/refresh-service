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

var chainIDs = map[string]int{
	"eth":            1,
	"eth:main":       1,
	"eth:goerli":     5,
	"eth:sepolia":    11155111,
	"polygon":        137,
	"polygon:main":   137,
	"polygon:mumbai": 80001,
	"zkevm":          1101,
	"zkevm:main":     1101,
	"zkevm:test":     1442,
}

type state struct {
	contracts map[int]*abi.State
}

func (s *state) verify(_ circuits.CircuitID, pubsignals []string) error {
	bytePubsig, err := json.Marshal(pubsignals)
	if err != nil {
		return err
	}

	authPubSignals := circuits.AuthV2PubSignals{}
	err = authPubSignals.PubSignalsUnmarshal(bytePubsig)
	if err != nil {
		return err
	}

	did, err := core.ParseDIDFromID(*authPubSignals.UserID)
	if err != nil {
		return err
	}

	id, err := core.IDFromDID(*did)
	if err != nil {
		return errors.WithStack(err)
	}

	blockchain, err := core.BlockchainFromID(id)
	if err != nil {
		return errors.WithStack(err)
	}

	networkID, err := core.NetworkIDFromID(id)
	if err != nil {
		return errors.WithStack(err)
	}

	chainID := chainIDs[fmt.Sprintf("%s:%s", blockchain, networkID)]
	contract, ok := s.contracts[chainID]
	if !ok {
		return errors.Errorf("not supported blockchain %s", blockchain)
	}

	globalState := authPubSignals.GISTRoot.BigInt()
	globalStateInfo, err := contract.GetGISTRootInfo(&bind.CallOpts{}, globalState)
	if err != nil {
		return err
	}
	if (big.NewInt(0)).Cmp(globalStateInfo.CreatedAtTimestamp) == 0 {
		return errors.Errorf("root %s doesn't exist in smart contract", globalState.String())
	}
	if globalState.Cmp(globalStateInfo.Root) != 0 {
		return errors.Errorf("invalid global state info in the smart contract, expected root %s, got %s", globalState.String(), globalStateInfo.Root.String())
	}

	if (big.NewInt(0)).Cmp(globalStateInfo.ReplacedByRoot) != 0 && time.Since(time.Unix(globalStateInfo.ReplacedAtTimestamp.Int64(), 0)) > time.Minute*15 {
		return errors.Errorf("global state is too old, replaced timestamp is %v", globalStateInfo.ReplacedAtTimestamp.Int64())
	}

	return nil
}

func NewPackageManager(
	supportedRPC map[string]string,
	supportedStateContracts map[string]string,
	circuitsFolderPath string,
) (*iden3comm.PackageManager, error) {
	circuitsPath := fmt.Sprintf("%s/%s", circuitsFolderPath, "authV2")
	_, err := os.Stat(fmt.Sprintf("%s/circuit_final.zkey", circuitsPath))
	if err != nil {
		return nil, errors.Errorf(
			"issuer with the file circuit_final.zkey by path '%s': %v", circuitsPath, err)
	}
	_, err = os.Stat(fmt.Sprintf("%s/circuit.wasm", circuitsPath))
	if err != nil {
		return nil, errors.Errorf(
			"issuer with the file circuit.wasm by path '%s': %v", circuitsPath, err)
	}
	verificationKey, err := os.ReadFile(fmt.Sprintf("%s/verification_key.json", circuitsPath))
	if err != nil {
		return nil, errors.Errorf(
			"issuer with the file verification_key.json by path '%s': %v", circuitsPath, err)
	}

	states := state{
		contracts: make(map[int]*abi.State, len(supportedStateContracts)),
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
