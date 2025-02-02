/*
* Copyright (C) 2020 The poly network Authors
* This file is part of The poly network library.
*
* The poly network is free software: you can redistribute it and/or modify
* it under the terms of the GNU Lesser General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The poly network is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Lesser General Public License for more details.
* You should have received a copy of the GNU Lesser General Public License
* along with The poly network . If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/polynetwork/poly/core/states"
	"github.com/polynetwork/poly/native/service/governance/neo3_state_manager"

	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Zilliqa/gozilliqa-sdk/account"
	"github.com/Zilliqa/gozilliqa-sdk/core"
	"github.com/Zilliqa/gozilliqa-sdk/crosschain/polynetwork"
	"github.com/Zilliqa/gozilliqa-sdk/provider"
	zilutil "github.com/Zilliqa/gozilliqa-sdk/util"

	"github.com/btcsuite/btcd/wire"
	types3 "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	common3 "github.com/ethereum/go-ethereum/common"
	"github.com/joeqian10/neo-gogogo/block"
	"github.com/joeqian10/neo-gogogo/helper/io"
	"github.com/joeqian10/neo-gogogo/rpc"

	block3 "github.com/joeqian10/neo3-gogogo/block"
	helper3 "github.com/joeqian10/neo3-gogogo/helper"
	io3 "github.com/joeqian10/neo3-gogogo/io"
	rpc3 "github.com/joeqian10/neo3-gogogo/rpc"
	"github.com/ontio/ontology-crypto/keypair"
	ontology_go_sdk "github.com/ontio/ontology-go-sdk"
	common2 "github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/smartcontract/service/native/cross_chain/header_sync"
	"github.com/ontio/ontology/smartcontract/service/native/governance"
	utils2 "github.com/ontio/ontology/smartcontract/service/native/utils"

	"github.com/polynetwork/eth-contracts/go_abi/eccm_abi"
	poly_go_sdk "github.com/polynetwork/poly-go-sdk"

	"github.com/polynetwork/kai-relayer/kaiclient"
	"github.com/polynetwork/poly-io-test/chains/btc"
	cosmos2 "github.com/polynetwork/poly-io-test/chains/cosmos"
	"github.com/polynetwork/poly-io-test/chains/eth"
	"github.com/polynetwork/poly-io-test/chains/ont"
	"github.com/polynetwork/poly-io-test/config"
	"github.com/polynetwork/poly-io-test/log"
	"github.com/polynetwork/poly-io-test/testcase"
	"github.com/polynetwork/poly/common"
	vconfig "github.com/polynetwork/poly/consensus/vbft/config"
	"github.com/polynetwork/poly/native/service/governance/node_manager"
	"github.com/polynetwork/poly/native/service/governance/relayer_manager"
	"github.com/polynetwork/poly/native/service/governance/side_chain_manager"
	"github.com/polynetwork/poly/native/service/header_sync/bsc"
	"github.com/polynetwork/poly/native/service/header_sync/cosmos"
	"github.com/polynetwork/poly/native/service/header_sync/heco"
	"github.com/polynetwork/poly/native/service/header_sync/msc"
	"github.com/polynetwork/poly/native/service/utils"
	"github.com/tendermint/tendermint/rpc/client/http"
	types2 "github.com/tendermint/tendermint/types"
)

var (
	tool                                                                  string
	toolConf                                                              string
	pWalletFiles                                                          string
	pPwds                                                                 string
	oWalletFiles                                                          string
	oPwds                                                                 string
	newWallet                                                             string
	newPwd                                                                string
	amt                                                                   int64
	keyFile                                                               string
	stateFile                                                             string
	id                                                                    uint64
	blockMsgDelay, hashMsgDelay, peerHandshakeTimeout, maxBlockChangeView uint64
	rootca                                                                string
	chainId                                                               uint64
	fabricRelayerTy                                                       uint64
	neo3StateValidators                                                   string
)

func init() {
	flag.StringVar(&tool, "tool", "", "choose a tool to run")
	flag.StringVar(&toolConf, "conf", "./config.json", "configuration file path")
	flag.StringVar(&pWalletFiles, "pwallets", "", "poly wallet files sep by ','")
	flag.StringVar(&pPwds, "ppwds", "", "poly pwd for every wallet, sep by ','")
	flag.StringVar(&oWalletFiles, "owallets", "", "ontology wallet files sep by ','")
	flag.StringVar(&oPwds, "opwds", "", "ontology pwd for every wallet, sep by ','")
	flag.StringVar(&newWallet, "newwallet", "", "new wallet adding to poly consensus")
	flag.StringVar(&newPwd, "newpwd", "", "password for new wallet")
	flag.Int64Var(&amt, "amt", 50, "amount to create new cosmos validator")
	flag.StringVar(&keyFile, "cosmos_val_privk_file", "", "cosmos validator's privk file")
	flag.StringVar(&stateFile, "cosmos_val_state_file", "", "cosmos validator's state file")
	flag.Uint64Var(&id, "id", 0, "chain id to quit")
	flag.Uint64Var(&blockMsgDelay, "blk_msg_delay", 5000, "")
	flag.Uint64Var(&hashMsgDelay, "hash_msg_delay", 5000, "")
	flag.Uint64Var(&peerHandshakeTimeout, "peer_handshake_timeout", 10, "")
	flag.Uint64Var(&maxBlockChangeView, "max_blk_change_view", 10000, "")
	flag.StringVar(&rootca, "rootca", "", "file path for root CA")
	flag.Uint64Var(&chainId, "chainid", 0, "default 0 means all chains")
	flag.Uint64Var(&fabricRelayerTy, "fab_relayer_type", 1, "the relayer of fabric type: how many orgs need to sign CA for relayer")

	flag.StringVar(&neo3StateValidators, "neo3_state_validators", "", "neo3 state root validator public keys in compressed format")

	flag.Parse()
}

func main() {
	log.InitLog(2, os.Stdout)

	err := config.DefConfig.Init(toolConf)
	if err != nil {
		panic(err)
	}
	poly := poly_go_sdk.NewPolySdk()
	if err := btc.SetUpPoly(poly, config.DefConfig.RchainJsonRpcAddress); err != nil {
		panic(err)
	}

	acc, err := btc.GetAccountByPassword(poly, config.DefConfig.RCWallet, []byte(config.DefConfig.RCWalletPwd))
	if err != nil {
		panic(err)
	}

	switch tool {
	case "register_side_chain":
		wArr := strings.Split(pWalletFiles, ",")
		pArr := strings.Split(pPwds, ",")

		accArr := make([]*poly_go_sdk.Account, len(wArr))
		for i, v := range wArr {
			accArr[i], err = btc.GetAccountByPassword(poly, v, []byte(pArr[i]))
			if err != nil {
				panic(fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, wArr[i], pArr[i]))
			}
		}
		switch chainId {
		case config.DefConfig.BtcChainID:
			if RegisterBtcChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.BtcChainID, poly, accArr)
			}
		case config.DefConfig.EthChainID:
			if RegisterEthChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.EthChainID, poly, accArr)
			}
		case config.DefConfig.OntChainID:
			if RegisterOntChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.OntChainID, poly, accArr)
			}
		case config.DefConfig.NeoChainID:
			if RegisterNeoChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.NeoChainID, poly, accArr)
			}
		case config.DefConfig.Neo3ChainID:
			if RegisterNeo3Chain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.Neo3ChainID, poly, accArr)
			}
		case config.DefConfig.CMCrossChainId:
			if RegisterCosmos(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.CMCrossChainId, poly, accArr)
			}
		case config.DefConfig.BscChainID:
			if RegisterBSC(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.BscChainID, poly, accArr)
			}
		case config.DefConfig.ZilChainID:
			if RegisterZIL(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.ZilChainID, poly, accArr)
			}
		case config.DefConfig.HecoChainID:
			if RegisterHeco(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.HecoChainID, poly, accArr)
			}
		case config.DefConfig.O3ChainID:
			if RegisterO3(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.O3ChainID, poly, accArr)
			}
		case config.DefConfig.MscChainID:
			if registerMSC(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.MscChainID, poly, accArr)
			}
		case config.DefConfig.OkChainID:
			if registerOK(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.OkChainID, poly, accArr)
			}
		case config.DefConfig.PolygonHeimdallChainID:
			if registerPolygonHeimdall(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.PolygonHeimdallChainID, poly, accArr)
			}
		case config.DefConfig.PolygonBorChainID:
			if registerPolygonBor(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.PolygonBorChainID, poly, accArr)
			}
		case config.DefConfig.KaiChainID:
			if RegisterKai(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.KaiChainID, poly, accArr)
			}
		case config.DefConfig.ArbChainID:
			if RegisterArb(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.ArbChainID, poly, accArr)
			}
		case config.DefConfig.XdaiChainID:
			if RegisterXdai(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.XdaiChainID, poly, accArr)
			}
		case 0:
			if RegisterBtcChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.BtcChainID, poly, accArr)
			}
			if RegisterOntChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.OntChainID, poly, accArr)
			}
			if RegisterEthChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.EthChainID, poly, accArr)
			}
			if RegisterCosmos(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.CMCrossChainId, poly, accArr)
			}
			if RegisterNeoChain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.NeoChainID, poly, accArr)
			}
			if RegisterNeo3Chain(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.Neo3ChainID, poly, accArr)
			}
			if RegisterBSC(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.BscChainID, poly, accArr)
			}
			if RegisterZIL(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.ZilChainID, poly, accArr)
			}
			if RegisterHeco(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.HecoChainID, poly, accArr)
			}
			if RegisterO3(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.O3ChainID, poly, accArr)
			}
			if registerMSC(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.MscChainID, poly, accArr)
			}
			if registerOK(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.OkChainID, poly, accArr)
			}
			if RegisterKai(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.KaiChainID, poly, accArr)
			}
			if registerPolygonHeimdall(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.PolygonHeimdallChainID, poly, accArr)
			}
			if registerPolygonBor(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.PolygonBorChainID, poly, accArr)
			}
			if RegisterArb(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.ArbChainID, poly, accArr)
			}
			if RegisterXdai(poly, acc) {
				ApproveRegisterSideChain(config.DefConfig.XdaiChainID, poly, accArr)
			}
		}
	case "sync_genesis_header":
		wArr := strings.Split(pWalletFiles, ",")
		pArr := strings.Split(pPwds, ",")

		accArr := make([]*poly_go_sdk.Account, len(wArr))
		for i, v := range wArr {
			accArr[i], err = btc.GetAccountByPassword(poly, v, []byte(pArr[i]))
			if err != nil {
				panic(fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, wArr[i], pArr[i]))
			}
		}

		switch chainId {
		case config.DefConfig.BtcChainID:
			SyncBtcGenesisHeader(poly, acc)
		case config.DefConfig.EthChainID:
			SyncEthGenesisHeader(poly, accArr)
		case config.DefConfig.OntChainID:
			SyncOntGenesisHeader(poly, accArr)
		case config.DefConfig.NeoChainID:
			SyncNeoGenesisHeader(poly, accArr)
		case config.DefConfig.Neo3ChainID:
			err = SyncNeo3GenesisHeader(poly, accArr)
			if err != nil {
				panic(err)
			}
		case config.DefConfig.CMCrossChainId:
			SyncCosmosGenesisHeader(poly, accArr)
		case config.DefConfig.BscChainID:
			SyncBSCGenesisHeader(poly, accArr)
		case config.DefConfig.ZilChainID:
			SyncZILGenesisHeader(poly, accArr)
		case config.DefConfig.HecoChainID:
			SyncHecoGenesisHeader(poly, accArr)
		case config.DefConfig.O3ChainID:
			SyncO3GenesisHeader(poly, accArr)
		case config.DefConfig.MscChainID:
			SyncMSCGenesisHeader(poly, accArr)
		case config.DefConfig.OkChainID:
			SyncOKGenesisHeader(poly, accArr)
		case config.DefConfig.KaiChainID:
			SyncKaiGenesisHeader(poly, accArr)
		case config.DefConfig.PolygonHeimdallChainID:
			SyncPolygonHeimdallGenesisHeader(poly, accArr)
		case config.DefConfig.PolygonBorChainID:
			SyncPolygonBorGenesisHeader(poly, accArr)
		case config.DefConfig.XdaiChainID:
			SyncXdaiGenesisHeader(poly, accArr)
		case 0:
			SyncBtcGenesisHeader(poly, acc)
			SyncEthGenesisHeader(poly, accArr)
			SyncOntGenesisHeader(poly, accArr)
			SyncCosmosGenesisHeader(poly, accArr)
			SyncNeoGenesisHeader(poly, accArr)
			SyncNeo3GenesisHeader(poly, accArr)
			SyncBSCGenesisHeader(poly, accArr)
			SyncZILGenesisHeader(poly, accArr)
			SyncHecoGenesisHeader(poly, accArr)
			SyncMSCGenesisHeader(poly, accArr)
			SyncOKGenesisHeader(poly, accArr)
			SyncKaiGenesisHeader(poly, accArr)
			SyncPolygonHeimdallGenesisHeader(poly, accArr)
			SyncXdaiGenesisHeader(poly, accArr)
		}

	case "update_btc":
		accArr := getPolyAccounts(poly)
		if UpdateBtc(poly, acc) {
			ApproveUpdateChain(config.DefConfig.BtcChainID, poly, accArr)
		}

	case "update_eth":
		accArr := getPolyAccounts(poly)
		if UpdateEth(poly, acc) {
			ApproveUpdateChain(config.DefConfig.EthChainID, poly, accArr)
		}

	case "update_neo":
		accArr := getPolyAccounts(poly)
		if UpdateNeo(poly, acc) {
			ApproveUpdateChain(config.DefConfig.NeoChainID, poly, accArr)
		}

	case "update_zil":
		accArr := getPolyAccounts(poly)
		if UpdateZil(poly, acc) {
			ApproveUpdateChain(config.DefConfig.ZilChainID, poly, accArr)
		}

	case "update_neo3":
		accArr := getPolyAccounts(poly)
		if UpdateNeo3(poly, acc) {
			ApproveUpdateChain(config.DefConfig.Neo3ChainID, poly, accArr)
		}

	case "update_ok":
		UpdateOK(poly, acc)
	case "update_bor":
		accArr := getPolyAccounts(poly)
		if UpdatePolygonBor(poly, acc) {
			ApproveUpdateChain(config.DefConfig.PolygonBorChainID, poly, accArr)
		}

	case "init_ont_acc":
		err := InitOntAcc()
		if err != nil {
			panic(err)
		}

	case "poly_add_node":
		accArr := getPolyAccounts(poly)
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		if RegisterCandidate(poly, acc) {
			ApproveCandidate(acc, poly, accArr)
		}
		CommitPolyDpos(poly, accArr)

	case "cosmos_create_validator":
		CosmosCreateValidator(keyFile, stateFile, amt)
	case "cosmos_delegate_validator":
		CosmosDelegateToVal(amt)
	case "black_poly_node":
		accArr := getPolyAccounts(poly)
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		BlackPolyNode(poly, acc, accArr)
	case "white_poly_node":
		accArr := getPolyAccounts(poly)
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		WhitePolyNode(poly, acc, accArr)
	case "quit_poly_node":
		accArr := getPolyAccounts(poly)
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		QuitNode(poly, acc)
		CommitPolyDpos(poly, accArr)
	case "get_poly_consensus":
		GetPolyConsensusInfo(poly)
	case "add_relayer":
		accArr := getPolyAccounts(poly)
		signer := acc
		acc = &poly_go_sdk.Account{}
		if newPwd != "" {
			acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
			if err != nil {
				panic(fmt.Errorf("failed to get new account: %v", err))
			}
		} else {
			acc.Address, _ = common.AddressFromBase58(newWallet)
		}
		ApproveRelayer(poly, RegisterRelayer(poly, acc, signer), accArr)
	case "register_state_validator":
		accArr := getPolyAccounts(poly)
		signer := acc
		svs := strings.Split(neo3StateValidators, ",")
		ApproveRegisterStateValidator(poly, RegisterStateValidator(poly, svs, signer), accArr)
	case "remove_state_validator":
		accArr := getPolyAccounts(poly)
		signer := acc
		svs := strings.Split(neo3StateValidators, ",")
		ApproveRemoveStateValidator(poly, RemoveStateValidator(poly, svs, signer), accArr)
	case "get_state_validator":
		GetStateValidator(poly)
	case "reg_poly_node":
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		RegisterCandidate(poly, acc)
	case "unreg_poly_node":
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		UnRegisterPolyCandidate(poly, acc)

	case "remove_relayer":
		accArr := getPolyAccounts(poly)
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		ApproveRemoveRelayer(poly, RemoveRelayer(poly, acc), accArr)
	case "get_relayer":
		acc, err = btc.GetAccountByPassword(poly, newWallet, []byte(newPwd))
		if err != nil {
			panic(fmt.Errorf("failed to get new account: %v", err))
		}
		GetRelayer(poly, acc)
	case "quit_side_chain":
		accArr := getPolyAccounts(poly)
		QuitSideChain(poly, id, acc)
		ApproveQuitSideChain(poly, id, accArr)
	case "get_side_chain":
		GetSideChain(poly, id)
	case "get_poly_config":
		GetPolyConfig(poly)
	case "update_poly_config":
		accArr := getPolyAccounts(poly)
		UpdatePolyConfig(poly, uint32(blockMsgDelay), uint32(hashMsgDelay), uint32(peerHandshakeTimeout),
			uint32(maxBlockChangeView), accArr)
	case "commit_poly_dpos":
		accArr := getPolyAccounts(poly)
		CommitPolyDpos(poly, accArr)
	case "commit_ont_dpos":
		CommitOntDpos()
	}
}

func getPolyAccounts(poly *poly_go_sdk.PolySdk) []*poly_go_sdk.Account {
	wArr := strings.Split(pWalletFiles, ",")
	pArr := strings.Split(pPwds, ",")
	accArr := make([]*poly_go_sdk.Account, len(wArr))
	var err error
	for i, v := range wArr {
		accArr[i], err = btc.GetAccountByPassword(poly, v, []byte(pArr[i]))
		if err != nil {
			panic(fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, wArr[i], pArr[i]))
		}
	}
	return accArr
}

func InitOntAcc() error {
	oi, err := ont.NewOntInvoker(config.DefConfig.OntJsonRpcAddress, config.DefConfig.OntContractsAvmPath,
		config.DefConfig.OntWallet, config.DefConfig.OntWalletPassword)
	if err != nil {
		return fmt.Errorf("failed to new ont invoker: %v", err)
	}

	ow := strings.Split(oWalletFiles, ",")
	op := strings.Split(oPwds, ",")
	oAccArr := make([]*ontology_go_sdk.Account, len(ow))
	pks := make([]keypair.PublicKey, len(ow))
	for i, v := range ow {
		oAccArr[i], err = ont.GetOntAccByPwd(v, op[i])
		if err != nil {
			return fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, ow[i], op[i])
		}
		pks[i] = oAccArr[i].PublicKey
	}

	multiSignAddr, _ := types.AddressFromBookkeepers(pks)
	tx, err := WithdrawONTFromConsensus(oi.OntSdk, 0, 20000000, oi.OntAcc, oAccArr, pks, multiSignAddr,
		oi.OntAcc.Address, 100000000)
	if err != nil {
		return err
	}
	oi.WaitTxConfirmation(tx)

	txHash, err := oi.OntSdk.Native.Ont.Transfer(0, 20000000, oi.OntAcc,
		oi.OntAcc, oi.OntAcc.Address, 100000000)
	if err != nil {
		return err
	}
	oi.WaitTxConfirmation(txHash)
	log.Infof("InitOntAcc, transfer ont to myself %s", oi.OntAcc.Address.ToBase58())

	for _, oa := range oAccArr {
		amount, err := oi.OntSdk.Native.Ong.BalanceOf(oa.Address)
		if err != nil {
			return err
		}
		tx, err := oi.OntSdk.Native.Ong.Transfer(0, 20000000, oa, oa, oi.OntAcc.Address, amount)
		if err != nil {
			return err
		}
		oi.WaitTxConfirmation(tx)
		log.Infof("InitOntAcc, Withdraw %d ONG to myself %s", amount, oi.OntAcc.Address.ToBase58())
	}

	return nil
}

func WithdrawONTFromConsensus(ontSdk *ontology_go_sdk.OntologySdk, gasPrice, gasLimit uint64,
	payer *ontology_go_sdk.Account, signers []*ontology_go_sdk.Account, pks []keypair.PublicKey, from common2.Address,
	to common2.Address, amount uint64) (common2.Uint256, error) {
	tx, err := ontSdk.Native.Ont.NewTransferTransaction(gasPrice, gasLimit, from, to, amount)
	if err != nil {
		return common2.UINT256_EMPTY, err
	}
	if payer != nil {
		ontSdk.SetPayer(tx, payer.Address)
		err = ontSdk.SignToTransaction(tx, payer)
		if err != nil {
			return common2.UINT256_EMPTY, err
		}
	}
	for _, singer := range signers {
		err = ontSdk.MultiSignToTransaction(tx, uint16((5*len(pks)+6)/7), pks, singer)
		if err != nil {
			return common2.UINT256_EMPTY, err
		}
	}

	return ontSdk.SendTransaction(tx)
}

func SyncBtcGenesisHeader(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) {
	cli := btc.NewRestCli(config.DefConfig.BtcRestAddr, config.DefConfig.BtcRestUser, config.DefConfig.BtcRestPwd)
	curr, _, err := cli.GetCurrentHeightAndHash()
	if err != nil {
		panic(fmt.Errorf("SyncBtcGenesisHeader failed: %v", err))
	}
	start := curr - curr%2016
	hdr, err := cli.GetHeader(start)
	if err != nil {
		panic(fmt.Errorf("SyncBtcGenesisHeader falied: %v", err))
	}
	var buf bytes.Buffer
	err = hdr.BtcEncode(&buf, wire.ProtocolVersion, wire.LatestEncoding)
	if err != nil {
		panic(err)
	}

	hb := make([]byte, 4)
	binary.BigEndian.PutUint32(hb, uint32(start))
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.BtcChainID, append(buf.Bytes(), hb...),
		[]*poly_go_sdk.Account{acc})
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("btc already synced")
		} else {
			panic(fmt.Errorf("SyncBtcGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		blkHash := hdr.BlockHash()
		log.Infof("successful to sync btc genesis header: ( height: %d, block_hash: %s, txhash: %s )", start,
			blkHash.String(), txhash.ToHexString())
	}
}

func SyncEthGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	tool := eth.NewEthTools(config.DefConfig.EthURL)
	curr, err := tool.GetNodeHeight()
	if err != nil {
		panic(err)
	}
	hdr, err := tool.GetBlockHeader(curr)
	if err != nil {
		panic(err)
	}
	raw, err := hdr.MarshalJSON()
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.EthChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("eth already synced")
		} else {
			panic(fmt.Errorf("SyncEthGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync eth genesis header: (height: %d, blk_hash: %s, txhash: %s )", curr,
			hdr.Hash().String(), txhash.ToHexString())
	}

	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.Eccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.ETHPrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncEthGenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("SyncEthGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	rawHdr := gB.Header.ToArray()

	contractabi, err := abi.JSON(strings.NewReader(eccm_abi.EthCrossChainManagerABI))
	if err != nil {
		log.Errorf("SyncEthGenesisHeader, abi.JSON error: %v", err)
		return
	}
	txData, err := contractabi.Pack("initGenesisBlock", rawHdr, publickeys)
	if err != nil {
		log.Errorf("SyncEthGenesisHeader, contractabi.Pack error: %v", err)
		return
	}

	eccm := common3.HexToAddress(config.DefConfig.Eccm)
	callMsg := ethereum.CallMsg{
		From: signer.Address, To: &eccm, Gas: 0, GasPrice: gasPrice,
		Value: big.NewInt(0), Data: txData,
	}
	gasLimit, err := tool.GetEthClient().EstimateGas(context.Background(), callMsg)
	if err != nil {
		log.Errorf("SyncEthGenesisHeader, estimate gas limit error: %s", err.Error())
		return
	}

	auth := testcase.MakeEthAuth(signer, nonce, gasPrice.Uint64(), gasLimit)
	tx, err := eccmContract.InitGenesisBlock(auth, rawHdr, publickeys)
	if err != nil {
		log.Errorf("SyncEthGenesisHeader, failed to sync poly header to ETH: %v", err)
		return
	}
	tool.WaitTransactionConfirm(tx.Hash())
	log.Infof("successful to sync poly genesis header to Ethereum: ( txhash: %s )", tx.Hash().String())
}

func SyncZILGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	type TxBlockAndDsComm struct {
		TxBlock *core.TxBlock
		DsBlock *core.DsBlock
		DsComm  []core.PairOfNode
	}

	zilSdk := provider.NewProvider(config.DefConfig.ZilURL)
	initDsComm, _ := zilSdk.GetCurrentDSComm()
	// as its name suggest, the tx epoch is actually a future tx block
	// zilliqa side has this limitation to avoid some risk that no tx block got mined yet
	nextTxEpoch, _ := strconv.ParseUint(initDsComm.CurrentTxEpoch, 10, 64)
	fmt.Printf("current tx block number is %s, ds block number is %s, number of ds guard is: %d\n", initDsComm.CurrentTxEpoch, initDsComm.CurrentDSEpoch, initDsComm.NumOfDSGuard)

	for {
		latestTxBlock, _ := zilSdk.GetLatestTxBlock()
		fmt.Println("wait current tx block got generated")
		latestTxBlockNum, _ := strconv.ParseUint(latestTxBlock.Header.BlockNum, 10, 64)
		fmt.Printf("latest tx block num is: %d, current tx block num is: %d", latestTxBlockNum, nextTxEpoch)
		if latestTxBlockNum >= nextTxEpoch {
			break
		}
		time.Sleep(time.Second * 20)
	}

	networkId, err := zilSdk.GetNetworkId()
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader failed: %s", err.Error()))
	}

	var dsComm []core.PairOfNode
	for _, ds := range initDsComm.DSComm {
		dsComm = append(dsComm, core.PairOfNode{
			PubKey: ds,
		})
	}

	dsBlockT, err := zilSdk.GetDsBlockVerbose(initDsComm.CurrentDSEpoch)
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader get ds block %s failed: %s", initDsComm.CurrentDSEpoch, err.Error()))
	}
	dsBlock := core.NewDsBlockFromDsBlockT(dsBlockT)

	txBlockT, err := zilSdk.GetTxBlockVerbose(initDsComm.CurrentTxEpoch)
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader get tx block %s failed: %s", initDsComm.CurrentTxEpoch, err.Error()))
	}

	txBlock := core.NewTxBlockFromTxBlockT(txBlockT)

	txBlockAndDsComm := TxBlockAndDsComm{
		TxBlock: txBlock,
		DsBlock: dsBlock,
		DsComm:  dsComm,
	}

	raw, err := json.Marshal(txBlockAndDsComm)
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader marshal genesis info failed: %s", err.Error()))
	}

	// sync zilliqa genesis info onto polynetwork
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.ZilChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("zil already synced")
		} else {
			panic(fmt.Errorf("SyncZILGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync zil genesis header, ds block: %s, tx block: %s, ds comm: %+v\n", zilutil.EncodeHex(dsBlock.BlockHash[:]), zilutil.EncodeHex(txBlock.BlockHash[:]), dsComm)
	}
	return
	// sycn poly network info to zilliqa cross chain manager
	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	genesisHeader := "0x" + zilutil.EncodeHex(gB.Header.ToArray())
	genesisPubKey := zilutil.EncodeHex(publickeys)

	publicKeys, err := polynetwork.SplitPubKeys(genesisPubKey)
	if err != nil {
		panic(err)
	}

	wallet := account.NewWallet()
	wallet.AddByPrivateKey(config.DefConfig.ZilPrivateKey)
	chainIdInt, _ := strconv.ParseInt(networkId, 10, 64)
	p := &polynetwork.Proxy{
		ProxyAddr:  config.DefConfig.ZilEccdProxy,
		ImplAddr:   config.DefConfig.ZilEccdImpl,
		Wallet:     wallet,
		Client:     zilSdk,
		ChainId:    int(chainIdInt),
		MsgVersion: 1,
	}

	tx, err := p.InitGenesisBlock(genesisHeader, publicKeys)
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader failed at init genesis block to zilliqa: %v", err))
	}

	hash, _ := tx.Hash()
	if tx.Receipt.Success {
		log.Infof("succeed to sync poly genesis header to ZIL: ( txhash: %s )", zilutil.EncodeHex(hash))
	} else {
		log.Infof("failed to sync poly genesis header to ZIL: ( txhash: %s )", zilutil.EncodeHex(hash))
	}

}

func SyncMSCGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	tool := eth.NewEthTools(config.DefConfig.MSCURL)
	height, err := tool.GetNodeHeight()
	if err != nil {
		panic(err)
	}

	epochHeight := height - height%200

	hdr, err := tool.GetBlockHeader(epochHeight)
	if err != nil {
		panic(err)
	}

	if len(hdr.Extra) <= 65+32 {
		panic(fmt.Sprintf("invalid epoch header at height:%d", epochHeight))
	}

	raw, err := json.Marshal(hdr)
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.MscChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("msc already synced")
		} else {
			panic(fmt.Errorf("SyncMSCGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync msc genesis header to poly: (height: %d, blk_hash: %s, txhash: %s )", epochHeight,
			hdr.Hash().String(), txhash.ToHexString())
	}

	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.MscEccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.MSCPrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncBSCGenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))
	auth := testcase.MakeEthAuth(signer, nonce, gasPrice.Uint64(), uint64(8000000))

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
	tool.WaitTransactionConfirm(tx.Hash())
	log.Infof("successful to sync poly genesis header to MSC: ( txhash: %s )", tx.Hash().String())
}

func SyncOKGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	tool := eth.NewEthTools(config.DefConfig.OKURL)

	rawHex, err := ioutil.ReadFile("../okex-verify/raw.hex")
	if err != nil {
		panic(fmt.Sprintf("ReadFile error:%v", err))
	}
	raw, err := hex.DecodeString(string(rawHex))
	if err != nil {
		panic(fmt.Sprintf("DecodeString error:%v", err))
	}

	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.OkChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("msc already synced")
		} else {
			panic(fmt.Errorf("SyncMSCGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync ok genesis header to poly: (txhash: %s )",
			txhash.ToHexString())
	}

	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.OkEccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.OKPrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncOKGenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))
	auth := testcase.MakeEthAuth(signer, nonce, gasPrice.Uint64(), uint64(8000000))

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
	if err != nil {
		log.Infof("fail to sync poly genesis header to OK: %v signer:%s", err, signer.Address.Hex())
		return
	}
	log.Infof("successful to sync poly genesis header to OK: ( txhash: %s )", tx.Hash().String())
}

func SyncO3GenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	tool := eth.NewEthTools(config.DefConfig.O3URL)
	height, err := tool.GetNodeHeight()
	if err != nil {
		panic(err)
	}

	epochHeight := height - height%200

	hdr, err := tool.GetBlockHeader(epochHeight)
	if err != nil {
		panic(err)
	}

	if len(hdr.Extra) <= 65+32 {
		panic(fmt.Sprintf("invalid epoch header at height:%d", epochHeight))
	}

	raw, err := json.Marshal(hdr)
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.O3ChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("o3 already synced")
		} else {
			panic(fmt.Errorf("SyncO3GenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync o3 genesis header to poly: (height: %d, blk_hash: %s, txhash: %s )", epochHeight,
			hdr.Hash().String(), txhash.ToHexString())
	}

	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.O3Eccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.O3PrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncO3GenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))
	auth := testcase.MakeEthAuth(signer, nonce, gasPrice.Uint64(), uint64(8000000))

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
	tool.WaitTransactionConfirm(tx.Hash())
	log.Infof("successful to sync poly genesis header to O3: ( txhash: %s )", tx.Hash().String())
}

func SyncHecoGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	tool := eth.NewEthTools(config.DefConfig.HecoURL)
	height, err := tool.GetNodeHeight()
	if err != nil {
		panic(err)
	}

	epochHeight := height - height%200
	pEpochHeight := epochHeight - 200

	hdr, err := tool.GetBlockHeader(epochHeight)
	if err != nil {
		panic(err)
	}
	phdr, err := tool.GetBlockHeader(pEpochHeight)
	if err != nil {
		panic(err)
	}
	pvalidators, err := heco.ParseValidators(phdr.Extra[32 : len(phdr.Extra)-65])
	if err != nil {
		panic(err)
	}

	if len(hdr.Extra) <= 65+32 {
		panic(fmt.Sprintf("invalid epoch header at height:%d", epochHeight))
	}
	if len(phdr.Extra) <= 65+32 {
		panic(fmt.Sprintf("invalid epoch header at height:%d", pEpochHeight))
	}

	genesisHeader := bsc.GenesisHeader{Header: *hdr, PrevValidators: []bsc.HeightAndValidators{
		{Height: big.NewInt(int64(pEpochHeight)), Validators: pvalidators},
	}}
	raw, err := json.Marshal(genesisHeader)
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.HecoChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("heco genesis header already synced")
		} else {
			panic(fmt.Errorf("SyncHecoGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync heco genesis header: (height: %d, blk_hash: %s, txhash: %s )", epochHeight,
			hdr.Hash().String(), txhash.ToHexString())
	}

	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.HecoEccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.HecoPrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncHecoGenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))
	auth := testcase.MakeEthAuth(signer, nonce, gasPrice.Uint64(), uint64(8000000))

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
	tool.WaitTransactionConfirm(tx.Hash())
	log.Infof("successful to sync poly genesis header to Heco: ( txhash: %s )", tx.Hash().String())
}

func SyncBSCGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	tool := eth.NewEthTools(config.DefConfig.BSCURL)
	height, err := tool.GetNodeHeight()
	if err != nil {
		panic(err)
	}

	epochHeight := height - height%200
	pEpochHeight := epochHeight - 200

	hdr, err := tool.GetBlockHeader(epochHeight)
	if err != nil {
		panic(err)
	}
	phdr, err := tool.GetBlockHeader(pEpochHeight)
	if err != nil {
		panic(err)
	}
	pvalidators, err := bsc.ParseValidators(phdr.Extra[32 : len(phdr.Extra)-65])
	if err != nil {
		panic(err)
	}

	if len(hdr.Extra) <= 65+32 {
		panic(fmt.Sprintf("invalid epoch header at height:%d", epochHeight))
	}
	if len(phdr.Extra) <= 65+32 {
		panic(fmt.Sprintf("invalid epoch header at height:%d", pEpochHeight))
	}

	genesisHeader := bsc.GenesisHeader{Header: *hdr, PrevValidators: []bsc.HeightAndValidators{
		{Height: big.NewInt(int64(pEpochHeight)), Validators: pvalidators},
	}}
	raw, err := json.Marshal(genesisHeader)
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.BscChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("bsc already synced")
		} else {
			panic(fmt.Errorf("SyncBSCGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync bsc genesis header: (height: %d, blk_hash: %s, txhash: %s )", epochHeight,
			hdr.Hash().String(), txhash.ToHexString())
	}

	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.BscEccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.BSCPrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncBSCGenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))
	chainId, err := tool.GetEthClient().ChainID(context.Background())
	if err != nil {
		panic(err)
	}
	//auth := testcase.MakeEthAuth(signer, nonce, gasPrice.Uint64(), uint64(8000000))
	auth := testcase.MakeEthAuthWithChainID(signer, nonce, gasPrice.Uint64(), uint64(8000000), chainId)

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
	if err != nil {
		panic(fmt.Errorf("commitGenesisHeader to bsc - InitGenesisBlock error: %s", err))
	}
	tool.WaitTransactionConfirm(tx.Hash())
	log.Infof("successful to sync poly genesis header to BSC: ( txhash: %s )", tx.Hash().String())
}

func SyncOntGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	ontCli := ontology_go_sdk.NewOntologySdk()
	ontCli.NewRpcClient().SetAddress(config.DefConfig.OntJsonRpcAddress)

	genesisBlock, err := ontCli.GetBlockByHeight(config.DefConfig.OntEpoch)
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.OntChainID, genesisBlock.Header.ToArray(), accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("ont already synced")
		} else {
			panic(fmt.Errorf("SyncOntGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync ont genesis header: ( txhash: %s )", txhash.ToHexString())
	}
	ow := strings.Split(oWalletFiles, ",")
	op := strings.Split(oPwds, ",")
	oAccArr := make([]*ontology_go_sdk.Account, len(ow))
	pks := make([]keypair.PublicKey, len(ow))
	for i, v := range ow {
		oAccArr[i], err = ont.GetOntAccByPwd(v, op[i])
		if err != nil {
			panic(fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, ow[i], op[i]))
		}
		pks[i] = oAccArr[i].PublicKey
	}
	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	txHash, err := InvokeNativeContractWithMultiSign(ontCli, 0, 2000000, pks, oAccArr, byte(0),
		utils2.HeaderSyncContractAddress, header_sync.SYNC_GENESIS_HEADER,
		[]interface{}{
			&header_sync.SyncGenesisHeaderParam{
				GenesisHeader: gB.Header.ToArray(),
			}})
	if err != nil {
		panic(fmt.Errorf("faild to sync poly header to ontology: %v", err))
	}
	ont.WaitOntTx(txHash, ontCli)
	log.Infof("successful to sync poly genesis header to Ontology: ( txhash: %s )", txHash.ToHexString())
}

func SyncNeoGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) error {
	cli := rpc.NewClient(config.DefConfig.NeoUrl)
	resp := cli.GetBlockHeaderByIndex(config.DefConfig.NeoEpoch)
	if resp.HasError() {
		return fmt.Errorf("failed to get header: %v", resp.Error.Message)
	}
	header, err := block.NewBlockHeaderFromRPC(&resp.Result)
	if err != nil {
		return err
	}
	buf := io.NewBufBinaryWriter()
	header.Serialize(buf.BinaryWriter)
	if buf.Err != nil {
		return buf.Err
	}

	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.NeoChainID, buf.Bytes(), accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("neo already synced")
		} else {
			panic(fmt.Errorf("SyncNeoGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync neo genesis header: ( txhash: %s )", txhash.ToHexString())
	}

	return nil
}

func SyncPolygonBorGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	// 16629815
	// headerBytes, _ := hex.DecodeString("7b22486561646572223a7b22706172656e7448617368223a22307831323561656135613563616232353836656531376638643535633832356138306236613632626237356430313334396366323466666337636562346638323733222c2273686133556e636c6573223a22307831646363346465386465633735643761616238356235363762366363643431616433313234353162393438613734313366306131343266643430643439333437222c226d696e6572223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030222c227374617465526f6f74223a22307837363532353836613039306336363034643365303331386136383436343037376133343933623934316366623130366138383532326539366561383865386633222c227472616e73616374696f6e73526f6f74223a22307838376161393065643564303632333939376463303863356132323361633266336339303231343930373466373834363132336332623630626131323638663666222c227265636569707473526f6f74223a22307838643661623264333539353333653735343664343330356462386433383237376564643331616363636536343131316438323039363833663631656332386236222c226c6f6773426c6f6f6d223a2230783030303030303030303030303030303630303030343030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303038303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303038303030303030383030303030303030303030303030303030303030313030303030303030303030303034303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303830303030303130303030303030303030303030303030303030303030303034303030303030303030303230303030303030303030303030303030303030303030303030303030303030303030303030323030303030303030303030303030303230303038303030303030303030303030303031303030303030303030303030303030303030303030303030303034303030303030303032303030303030303030303031303030303030303030303030303030303030303030303030303030303030313030303030303030303030303030383030303030303030303031303030303031303030303030303030303030303030303030303030303030303030303030303030303030303030313030303030222c22646966666963756c7479223a22307839222c226e756d626572223a223078666463303337222c226761734c696d6974223a22307831333132643030222c2267617355736564223a22307838373932222c2274696d657374616d70223a2230783630663832353536222c22657874726144617461223a2230786437383330313061303338333632366637323838363736663331326533313335326533353835366336393665373537383030303030303030303030303030303063316666343661646434316137666237396463353334626362363834643632366635393830313733353866373039613735346238316132633036303937613437363336356531356432333766653838633163323735356533323861643935376630656135643866666563333666333963643337333765303533353834373731333031222c226d697848617368223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030222c226e6f6e6365223a22307830303030303030303030303030303030222c2268617368223a22307865653034353765373535326437663139656335666439666238363836653464343239313433663264663736316234656166663764353065633339636430636435227d2c22536e617073686f74223a7b2268617368223a22307865653034353765373535326437663139656335666439666238363836653464343239313433663264663736316234656166663764353065633339636430636435222c2276616c696461746f72536574223a7b2276616c696461746f7273223a5b7b224944223a302c227369676e6572223a22307839326461396638663365653136613237363839366663376232353530623231353161616530333332222c22706f776572223a33373839352c22616363756d223a2d3239343133327d2c7b224944223a302c227369676e6572223a22307862323663323232333738313664383938636239393932643736373434343130356266646330336236222c22706f776572223a343335332c22616363756d223a3631343536377d2c7b224944223a302c227369676e6572223a22307862653138386436363431653862363830373433613438313564666130663632303830333839363066222c22706f776572223a3330343334332c22616363756d223a2d313135393137307d2c7b224944223a302c227369676e6572223a22307863323638383061306166326561306337653831333065366563343761663735363436353435326538222c22706f776572223a313034313731322c22616363756d223a2d3535333533397d2c7b224944223a302c227369676e6572223a22307863323735646338626533396635306431326636366236613633363239633339646135626165356264222c22706f776572223a313032343633332c22616363756d223a313232303138337d2c7b224944223a302c227369676e6572223a22307863343433323739613636323830666139626232393136393939633563326432666163616230353739222c22706f776572223a373030302c22616363756d223a2d3131313836307d2c7b224944223a302c227369676e6572223a22307863346163663866626532383239636230633230396466663135613938623364633133663132623166222c22706f776572223a3136302c22616363756d223a3636393530317d2c7b224944223a302c227369676e6572223a22307865346238653932323237303434303161643136643464383236373332393533646166303763376532222c22706f776572223a3430313730372c22616363756d223a2d3335323131367d2c7b224944223a302c227369676e6572223a22307866393033626139653030363139336331353237626662653635666532313233373034656133663939222c22706f776572223a31323235382c22616363756d223a2d33333433337d5d2c2270726f706f736572223a7b224944223a302c227369676e6572223a22307863323638383061306166326561306337653831333065366563343761663735363436353435326538222c22706f776572223a313034313731322c22616363756d223a2d3535333533397d7d7d7d")
	// txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.PolygonBorChainID, headerBytes, accArr)
	// if err != nil {
	// 	if strings.Contains(err.Error(), "had been initialized") {
	// 		log.Info("heimdall already synced")
	// 	} else {
	// 		panic(err)
	// 	}
	// } else {
	// 	testcase.WaitPolyTx(txhash, poly)
	// 	log.Infof("successful to sync bor genesis header: ( txhash: %s )", txhash.ToHexString())
	// }

	tool := eth.NewEthTools(config.DefConfig.BorURL)
	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.BorEccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.BorPrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncPolygonBorGenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))
	auth := testcase.MakeEthAuthWithChainID(signer, nonce, gasPrice.Uint64(), uint64(800000), big.NewInt(int64(config.DefConfig.PolygonBorSignerChainID)))

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
	if err != nil {
		panic(fmt.Sprintf("InitGenesisBlock failed:%v", err))
	}

	tool.WaitTransactionConfirm(tx.Hash())
	log.Infof("successful to sync poly genesis header to bor: ( txhash: %s )", tx.Hash().String())
}

func SyncXdaiGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	tool := eth.NewEthTools(config.DefConfig.XdaiURL)
	eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.XdaiEccm), tool.GetEthClient())
	if err != nil {
		panic(err)
	}
	signer, err := eth.NewEthSigner(config.DefConfig.XdaiPrivateKey)
	if err != nil {
		panic(err)
	}
	nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
	gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
	if err != nil {
		panic(fmt.Errorf("SyncXdaiGenesisHeader, get suggest gas price failed error: %s", err.Error()))
	}
	gasPrice = gasPrice.Mul(gasPrice, big.NewInt(1))
	chainId, err := tool.GetEthClient().ChainID(context.Background())
	if err != nil {
		panic(err)
	}
	auth := testcase.MakeEthAuthWithChainID(signer, nonce, gasPrice.Uint64(), uint64(0), chainId)

	gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	if err != nil {
		panic(err)
	}
	info := &vconfig.VbftBlockInfo{}
	if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
		panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
	}

	var bookkeepers []keypair.PublicKey
	for _, peer := range info.NewChainConfig.Peers {
		keystr, _ := hex.DecodeString(peer.ID)
		key, _ := keypair.DeserializePublicKey(keystr)
		bookkeepers = append(bookkeepers, key)
	}
	bookkeepers = keypair.SortPublicKeys(bookkeepers)

	publickeys := make([]byte, 0)
	for _, key := range bookkeepers {
		publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	}

	tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
	if err != nil {
		panic(fmt.Sprintf("InitGenesisBlock failed:%v", err))
	}

	tool.WaitTransactionConfirm(tx.Hash())
	log.Infof("successful to sync poly genesis header to xdai: ( txhash: %s )", tx.Hash().String())
}

func SyncPolygonHeimdallGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	// 6504476
	headerBytes, _ := hex.DecodeString("0ab5020a02080a120e6865696d64616c6c2d3830303031189c808d03220b08a6c6e087061092d4983530a09d0c3a480a205611470b348b23c97d3a3e238e54a8305b026e605e9b564438fa8d9d9db79f291224080112204e1150cc9200124748a0616e349cd255db454d624690a680270f542a08c0a0684220bb725c12397f2348e73f3bad84c1941b5246226085551e00543c9fc17608e91c5220120d78d991fbb32e771c8bb435f457c08b70cbe32ae718e219963dc27bd536235a20120d78d991fbb32e771c8bb435f457c08b70cbe32ae718e219963dc27bd53623622081ba6261d0077795e489737675de120cc9170adccaad805e12ef2708a2e214536a208b4395dd3b8893c592609a5e0065ae645b888ff91f6568f942890b3e948b69e8820114c275dc8be39f50d12f66b6a63629c39da5bae5bd12f70a0a480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f12b8010802109c808d0322480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f2a0c08abc6e0870610a3b6c8f802321492da9f8f3ee16a276896fc7b2550b2151aae03324241aca8e5c3440e64c2b07d10ebaed615ddc1127353a0c81c2834232024e1eecbc93230799a2ae4635cae4705a16972314b276619831b8fa6d062329521239488ea0112ba010802109c808d0322480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f2a0c08abc6e0870610cab79693023214b26c22237816d898cb9992d767444105bfdc03b638014241ad27fdf4b2ea6a72a81d8ef7b21e29d6556d25f5234a423207685f54761bc1ba35dbce62ee31125eb77946408b0d0effc2b87d7cb71757d39275ce31528c4a7e0012ba010802109c808d0322480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f2a0c08abc6e0870610e5f7d392023214be188d6641e8b680743a4815dfa0f6208038960f38024241b7673c905ae8999b8a5d9b6a6809190820723a7c317acd1265034bc9cebbbcd00f2e10121861ec4a62888bba13dfab602dcb105117b8bfed06991b94ed0508190012ba010802109c808d0322480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f2a0c08abc6e0870610f6c295ed013214c26880a0af2ea0c7e8130e6ec47af756465452e8380342412a79f8c170eef476f61c54726ed3178f394e6b9688d53edd0f50874b4d8a3a45363c83ca526539c012542ee4c742aa2c362df97c1c8e93d27e9fc00118c2619d0112ba010802109c808d0322480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f2a0c08abc6e0870610c89de19c023214c275dc8be39f50d12f66b6a63629c39da5bae5bd38044241e6005e1a55a3f23e4289222427227ce424d6a83dd07dfc6d780bdbaa2b0100e93f4f4f8a001a5792c9af29e0564d3da52a78a7854a5de506d5dc5a999b9e73b8011200120012ba010802109c808d0322480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f2a0c08abc6e087061097f4c89b033214e4b8e9222704401ad16d4d826732953daf07c7e238074241227a0e3b398e5d1276ef2e0f7566ad0698a57a57995b342c17bb709f5cb4a4a648ca1c7cee87baf37989281097d9cb59e324f1a66b0ba868b2e687ee9e565cbe0012ba010802109c808d0322480a2041354bc7403d16716c952d2f430c3c4c482976eb47c33483ad3f9ad85e057d89122408011220b4ceeef38025effe829a1246dbefc38f6248267b9ae9d3deaabce1436ce5a42f2a0c08abc6e0870610ca82e992023214f903ba9e006193c1527bfbe65fe2123704ea3f9938084241adbf4a66722d580abfe2fccae7088add0204e20ff4265a314a2141c929e3990d58a2dcd6cddcc1c268089c44ee39979fbd16c28d684facb71d31372e1faef7f4001a660a1492da9f8f3ee16a276896fc7b2550b2151aae03321246eb5ae98741041f2c0ff8f11c0584bad20b3d275a025f567deda7b8ec97600509398cceba1f3649fc8b424b4754032980770a4c495706d5191d051e6423d5b8e63cd7792aa3d51887a80220d7b7151a6c0a14b26c22237816d898cb9992d767444105bfdc03b61246eb5ae98741047ae11341f861697b349afdae3c7328ced67fcfc82da84f8db22c78e229edcdcfd83a5bb7d69a56b352f95eadcb51b98be15d140f06c85c9cb414dfe317df4b7418812220fcc0c0ffffffffffff011a660a14be188d6641e8b680743a4815dfa0f6208038960f1246eb5ae9874104888a737a003f4e522ccf23bd9980fdbe7ef2b54365249deba0f9acd45279d66355b1864173b2cf9e75a1cbfb45e65a1a72b9ea76e47aa4bd50d79772ef30176918d7c91220e4ce571a660a14c26880a0af2ea0c7e8130e6ec47af756465452e81246eb5ae98741040bec8102c221c7cfff3e250bb6cc01c3b9a3964fb1bf4d53e91905320eef09595acb09ee0950e7374ec19488ff2523f186f6b1a9164c78dba8602e4e3c4eb01318b0ca3f20e1a8121a6d0a14c275dc8be39f50d12f66b6a63629c39da5bae5bd1246eb5ae9874104f3f18a027c929380417d2bd7d2a489cb662d4977e9daff335bc51f23c1c5f5f468aa19c6c8e937a745462ef2550bce42e4f38608dffb5a06e7b9d27d964cffee18f9c43e20b5d1edffffffffffff011a650a14c443279a66280fa9bb2916999c5c2d2facab05791246eb5ae98741046e58afa78fade1229ce3bebe3ed5435d895cfdc399323d4f20752935ff04dc514e8f3320a8d5434a13acc9209b9657ebbdf154ae715830135997f6c2ae02825818d8362097951e1a6c0a14c4acf8fbe2829cb0c209dff15a98b3dc13f12b1f1246eb5ae9874104161cf579b40ea1a68f166da216c50e88f1323213cd22a8ffa6acabc45893a80250b5aafa6dea6e4a0289ebabe8b2996ae806098b7d88d2eee8634ec73fe2edfd18a00120a2989cffffffffffff011a660a14e4b8e9222704401ad16d4d826732953daf07c7e21246eb5ae987410469bd14dadd683cb4a4d1e27b79d3594c2025716abaf3a8a8282b126ea5c3a686071033ef6aa4c9b7d12efb957a7a55faaa5684653895d25e88199e4c5281dffc18f7e61820e7ab2c1a6c0a14f903ba9e006193c1527bfbe65fe2123704ea3f991246eb5ae9874104dcd2883416e7b8663caafbfc885e757b0ea809657df8d6f322f01a0c5a11fd033bf13d3e0d5e88feff92ba415d32d626e3f7d9dd7b5ec7c2fef8ded83d660ac218e25f2098e5ebffffffffffff01")
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.PolygonHeimdallChainID, headerBytes, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("heimdall already synced")
		} else {
			panic(err)
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync heimdall genesis header: ( txhash: %s )", txhash.ToHexString())
	}
}

func SyncKaiGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	client, err := kaiclient.Dial(config.DefConfig.KaiUrl)
	if err != nil {
		panic(err)
	}

	header, err := client.FullHeaderByNumber(context.Background(), big.NewInt(config.DefConfig.KaiEpoch))
	if err != nil {
		panic(err)
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.KaiChainID, headerBytes, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("kai already synced")
		} else {
			panic(err)
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync kai genesis header: ( txhash: %s )", txhash.ToHexString())
	}

}

func SyncNeo3GenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) error {
	cli := rpc3.NewClient(config.DefConfig.Neo3Url)
	resp := cli.GetBlockHeader(strconv.Itoa(int(config.DefConfig.Neo3Epoch)))
	if resp.HasError() {
		return fmt.Errorf("failed to get header: %v", resp.Error.Message)
	}
	header, err := block3.NewBlockHeaderFromRPC(&resp.Result)
	if err != nil {
		return err
	}
	buf := io3.NewBufBinaryWriter()
	header.Serialize(buf.BinaryWriter)
	if buf.Err != nil {
		return buf.Err
	}
	//// concatenate state validator to genesis header bytes
	//sv, err := crypto3.AddressToScriptHash(config.DefConfig.Neo3StateValidator, helper3.DefaultAddressVersion)
	//if err != nil {
	//	return err
	//}
	//extra := append([]byte{0x14}, sv.ToByteArray()...)
	//bs := buf.Bytes()
	//bs = append(bs, extra...)

	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.Neo3ChainID, buf.Bytes(), accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("neo3 already synced")
		} else {
			panic(fmt.Errorf("SyncNeo3GenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync neo3 genesis header: ( txhash: %s )", txhash.ToHexString())
	}

	//// start sync poly header to neo3 side chain
	//if config.DefConfig.Neo3Wallet == "" {
	//	log.Infof("neo3 wallet empty, won't sync poly header to neo3 ccmc")
	//	return nil
	//}
	//polyHeader, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
	//if err != nil {
	//	return fmt.Errorf("poly.GetHeader(%d) to be synced to neo3 as genesis err: %v", config.DefConfig.RCEpoch, err)
	//}
	//
	//cp1 := sc.ContractParameter{
	//	Type:  sc.ByteArray,
	//	Value: polyHeader.Header.GetMessage(),
	//}
	//// public keys
	//info := &vconfig.VbftBlockInfo{}
	//if err := json.Unmarshal(polyHeader.Header.ConsensusPayload, info); err != nil {
	//	return fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err)
	//}
	//var bookkeepers []keypair.PublicKey
	//for _, peer := range info.NewChainConfig.Peers {
	//	keystr, _ := hex.DecodeString(peer.ID)
	//	key, _ := keypair.DeserializePublicKey(keystr)
	//	bookkeepers = append(bookkeepers, key)
	//}
	//bookkeepers = keypair.SortPublicKeys(bookkeepers)
	//publickeys := make([]byte, 0)
	//for _, key := range bookkeepers {
	//	publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
	//}
	//cp2 := sc.ContractParameter{
	//	Type:  sc.ByteArray,
	//	Value: publickeys,
	//}
	//
	//invoker, err := neo3.NewNeo3Invoker()
	//if err != nil {
	//	return fmt.Errorf("NewNeo3Invoker err: %v", err)
	//}
	//// build script
	//scriptHash, err := helper3.UInt160FromString(config.DefConfig.Neo3CCMC) // big endian
	//if err != nil {
	//	return fmt.Errorf("neo3 ccmc conversion error: %s", err)
	//}
	//
	//script, err := sc3.MakeScript(scriptHash, "InitGenesisBlock", []interface{}{cp1, cp2})
	//if err != nil {
	//	return fmt.Errorf("neo3 sc.MakeScript error: %s", err)
	//}
	//
	//balancesGas, err := invoker.GetAccountAndBalance(tx3.GasToken)
	//if err != nil {
	//	return fmt.Errorf("neo3 GetAccountAndBalance error: %s", err)
	//}
	//
	//// make transaction
	//trx, err := invoker.MakeTransaction(script, nil, []tx3.ITransactionAttribute{}, balancesGas)
	//if err != nil {
	//	return fmt.Errorf("neo3 MakeTransaction error: %s", err)
	//}
	//
	//// sign transaction
	//trx, err = invoker.SignTransaction(trx, config.DefConfig.Neo3Magic)
	//if err != nil {
	//	return fmt.Errorf("neo3 SignTransaction error: %s", err)
	//}
	//rawTxString := crypto3.Base64Encode(trx.ToByteArray())
	//
	//// send the raw transaction
	//response := invoker.Client.SendRawTransaction(rawTxString)
	//if response.HasError() {
	//	return fmt.Errorf("initGenesisBlock on neo3, SendRawTx err: %v", err)
	//}
	//log.Infof("sync poly header to neo3 as genesis, neo3TxHash: %s", trx.GetHash().String())
	//
	return nil
}

func SyncCosmosGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {

	invoker, err := cosmos2.NewCosmosInvoker()
	if err != nil {
		panic(err)
	}
	res, err := invoker.RpcCli.Commit(&config.DefConfig.CMEpoch)
	if err != nil {
		panic(err)
	}
	vals, err := getValidators(invoker.RpcCli, config.DefConfig.CMEpoch)
	if err != nil {
		panic(err)
	}
	ch := &cosmos.CosmosHeader{
		Header:  *res.Header,
		Commit:  res.Commit,
		Valsets: vals,
	}
	raw, err := invoker.CMCdc.MarshalBinaryBare(ch)
	if err != nil {
		panic(err)
	}
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.CMCrossChainId, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("cosmos already synced")
		} else {
			panic(err)
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync cosmos genesis header: ( txhash: %s )", txhash.ToHexString())
	}

	if config.DefConfig.CMChainId != "okexchain" {
		header, err := poly.GetHeaderByHeight(config.DefConfig.RCEpoch)
		if err != nil {
			panic(err)
		}

		tx, err := invoker.SyncPolyGenesisHdr(invoker.Acc.Acc, header.ToArray())
		if err != nil {
			panic(err)
		}

		log.Infof("successful to sync poly genesis header to cosmos: ( txhash: %s )", tx.Hash.String())
	} else {
		tool := eth.NewEthTools(config.DefConfig.OKURL)
		eccmContract, err := eccm_abi.NewEthCrossChainManager(common3.HexToAddress(config.DefConfig.OkEccm), tool.GetEthClient())
		if err != nil {
			panic(err)
		}
		signer, err := eth.NewEthSigner(config.DefConfig.OKPrivateKey)
		if err != nil {
			panic(err)
		}
		nonce := eth.NewNonceManager(tool.GetEthClient()).GetAddressNonce(signer.Address)
		gasPrice, err := tool.GetEthClient().SuggestGasPrice(context.Background())
		if err != nil {
			panic(fmt.Errorf("SyncCosmosGenesisHeader, get suggest gas price failed error: %s", err.Error()))
		}
		gasPrice = gasPrice.Mul(gasPrice, big.NewInt(5))
		auth := testcase.MakeEthAuth(signer, nonce, gasPrice.Uint64(), uint64(8000000))

		gB, err := poly.GetBlockByHeight(config.DefConfig.RCEpoch)
		if err != nil {
			panic(err)
		}
		info := &vconfig.VbftBlockInfo{}
		if err := json.Unmarshal(gB.Header.ConsensusPayload, info); err != nil {
			panic(fmt.Errorf("commitGenesisHeader - unmarshal blockInfo error: %s", err))
		}

		var bookkeepers []keypair.PublicKey
		for _, peer := range info.NewChainConfig.Peers {
			keystr, _ := hex.DecodeString(peer.ID)
			key, _ := keypair.DeserializePublicKey(keystr)
			bookkeepers = append(bookkeepers, key)
		}
		bookkeepers = keypair.SortPublicKeys(bookkeepers)

		publickeys := make([]byte, 0)
		for _, key := range bookkeepers {
			publickeys = append(publickeys, ont.GetOntNoCompressKey(key)...)
		}

		tx, err := eccmContract.InitGenesisBlock(auth, gB.Header.ToArray(), publickeys)
		tool.WaitTransactionConfirm(tx.Hash())

		log.Infof("successful to sync poly genesis header to cosmos: ( txhash: %s )", tx.Hash().String())
	}
}

func getValidators(rpc *http.HTTP, h int64) ([]*types2.Validator, error) {
	p := 1
	vSet := make([]*types2.Validator, 0)
	for {
		res, err := rpc.Validators(&h, p, 100)
		if err != nil {
			if strings.Contains(err.Error(), "page should be within") {
				return vSet, nil
			}
			return nil, err
		}
		// In case tendermint don't give relayer the right error
		if len(res.Validators) == 0 {
			return vSet, nil
		}
		vSet = append(vSet, res.Validators...)
		p++
	}
}

func InvokeNativeContractWithMultiSign(
	sdk *ontology_go_sdk.OntologySdk,
	gasPrice,
	gasLimit uint64,
	pubKeys []keypair.PublicKey,
	singers []*ontology_go_sdk.Account,
	cversion byte,
	contractAddress common2.Address,
	method string,
	params []interface{},
) (common2.Uint256, error) {
	tx, err := sdk.Native.NewNativeInvokeTransaction(gasPrice, gasLimit, cversion, contractAddress, method, params)
	if err != nil {
		return common2.UINT256_EMPTY, err
	}
	for _, singer := range singers {
		err = sdk.MultiSignToTransaction(tx, uint16((5*len(pubKeys)+6)/7), pubKeys, singer)
		if err != nil {
			return common2.UINT256_EMPTY, err
		}
	}
	return sdk.SendTransaction(tx)
}

func ApproveRegisterSideChain(id uint64, poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, a := range accArr {
		txhash, err = poly.Native.Scm.ApproveRegisterSideChain(id, a)
		if err != nil {
			panic(fmt.Errorf("No%d ApproveRegisterSideChain failed: %v", i, err))
		}
		log.Infof("No%d: successful to approve: ( acc: %s, txhash: %s, chain-id: %d )", i, a.Address.ToBase58(), txhash.ToHexString(), id)
	}
	testcase.WaitPolyTx(txhash, poly)
}

func RegisterBtcChain(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	var tyNet utils.BtcNetType
	switch config.BtcNet.Name {
	case "testnet3":
		blkToWait = 6
		tyNet = utils.TyTestnet3
	case "mainnet":
		blkToWait = 6
		tyNet = utils.TyMainnet
	case "regtest":
		tyNet = utils.TyRegtest
	case "simnet":
		tyNet = utils.TySimnet
	}

	rawTy := make([]byte, 8)
	binary.LittleEndian.PutUint64(rawTy, uint64(tyNet))
	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.BtcChainID, config.DefConfig.BtcChainID, "btc",
		blkToWait, rawTy, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("btc chain %d already registered", config.DefConfig.BtcChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("btc chain %d already requested", config.DefConfig.BtcChainID)
			return true
		}
		panic(fmt.Errorf("RegisterBtcChain failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register btc chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterKai(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.KaiEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterKai, failed to decode eccd '%s' : %v", config.DefConfig.KaiEccd, err))
	}
	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.KaiChainID, 13, "kai",
		blkToWait, eccd, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("eth chain %d already registered", config.DefConfig.KaiChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("eth chain %d already requested", config.DefConfig.KaiChainID)
			return true
		}
		panic(fmt.Errorf("RegisterKai failed: %v", err))
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register eth chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterEthChain(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	if config.BtcNet.Name == "testnet3" {
		blkToWait = 12
	}
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.Eccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterEthChain, failed to decode eccd '%s' : %v", config.DefConfig.Eccd, err))
	}
	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.EthChainID, 2, "eth",
		blkToWait, eccd, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("eth chain %d already registered", config.DefConfig.EthChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("eth chain %d already requested", config.DefConfig.EthChainID)
			return true
		}
		panic(fmt.Errorf("RegisterEthChain failed: %v", err))
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register eth chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterNeoChain(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	eccd, err := common2.AddressFromHexString(strings.TrimPrefix(config.DefConfig.NeoCCMC, "0x"))
	if err != nil {
		panic(fmt.Errorf("RegisterNeoChain, failed to decode eccd '%s' : %v", config.DefConfig.Eccd, err))
	}
	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.NeoChainID, 4, "NEO",
		blkToWait, eccd[:], acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("neo chain %d already registered", config.DefConfig.NeoChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("neo chain %d already requested", config.DefConfig.NeoChainID)
			return true
		}
		panic(fmt.Errorf("RegisterNeoChain failed: %v", err))
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register neo chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterNeo3Chain(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	neo3Ccmc := helper3.HexToBytes(config.DefConfig.Neo3CCMC)
	if len(neo3Ccmc) != 4 {
		panic(fmt.Errorf("incorrect Neo3CCMC length"))
	}
	txHash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.Neo3ChainID, 14, "NEO3",
		blkToWait, neo3Ccmc[:], helper3.UInt32ToBytes(config.DefConfig.Neo3Magic), acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("neo3 chain %d already registered", config.DefConfig.Neo3ChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("neo3 chain %d already requested", config.DefConfig.Neo3ChainID)
			return true
		}
		panic(fmt.Errorf("RegisterNeo3Chain failed: %v", err))
	}
	testcase.WaitPolyTx(txHash, poly)
	log.Infof("successful to register neo3 chain, txHash: %s", txHash.ToHexString())

	return true
}

func RegisterCosmos(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.CMCrossChainId, 5, "switcheochain",
		blkToWait, []byte{}, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("Cosmos chain %d already registered", config.DefConfig.CMCrossChainId)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("Cosmos chain %d already requested", config.DefConfig.CMCrossChainId)
			return true
		}
		panic(fmt.Errorf("RegisterCosmosChain failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register cosmos chain: ( txhash: %s )", txhash.ToHexString())

	return true

}

func registerMSC(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	tool := eth.NewEthTools(config.DefConfig.MSCURL)
	chainID, err := tool.GetChainID()
	if err != nil {
		panic(err)
	}

	blkToWait := uint64(15)
	extra := msc.ExtraInfo{
		ChainID: chainID,
		Period:  15,
		Epoch:   30000,
	}

	extraBytes, _ := json.Marshal(extra)

	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.MscEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("registerMSC, failed to decode eccd '%s' : %v", config.DefConfig.MscEccd, err))
	}

	fmt.Println("config.DefConfig.MSCChainID", config.DefConfig.MscChainID, "extraBytes", string(extraBytes), "MscEccd", config.DefConfig.MscEccd)
	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.MscChainID, 10, "msc",
		blkToWait, eccd, extraBytes, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("bsc chain %d already registered", config.DefConfig.MscChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("bsc chain %d already requested", config.DefConfig.MscChainID)
			return true
		}
		panic(fmt.Errorf("registerMSC failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register msc chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

type PolygonExtraInfo struct {
	Sprint              uint64
	Period              uint64
	ProducerDelay       uint64
	BackupMultiplier    uint64
	HeimdallPolyChainID uint64
}

func UpdatePolygonBor(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(128)
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.BorEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("registerPolygonBor, failed to decode eccd '%s' : %v", config.DefConfig.BorEccd, err))
	}

	heimdallPolyChainID := uint64(config.DefConfig.PolygonHeimdallChainID)

	extra := PolygonExtraInfo{
		Sprint:              64,
		Period:              2,
		ProducerDelay:       6,
		BackupMultiplier:    2,
		HeimdallPolyChainID: heimdallPolyChainID,
	}
	extraBytes, _ := json.Marshal(extra)

	if err := updateSideChainExt(poly, acc, config.DefConfig.PolygonBorChainID, 16, blkToWait, "bor", eccd, extraBytes); err != nil {
		log.Errorf("failed to update bor: %v", err)
		return false
	}
	return true
}

func registerPolygonBor(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(128)
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.BorEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("registerPolygonBor, failed to decode eccd '%s' : %v", config.DefConfig.BorEccd, err))
	}

	heimdallPolyChainID := uint64(config.DefConfig.PolygonHeimdallChainID)

	extra := PolygonExtraInfo{
		Sprint:              64,
		Period:              2,
		ProducerDelay:       6,
		BackupMultiplier:    2,
		HeimdallPolyChainID: heimdallPolyChainID,
	}
	extraBytes, _ := json.Marshal(extra)

	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.PolygonBorChainID, 16, "bor",
		blkToWait, eccd, extraBytes, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("chain %d already registered", config.DefConfig.OkChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("chain %d already requested", config.DefConfig.OkChainID)
			return true
		}
		panic(fmt.Errorf("registerPolygonBor failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register bor chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func registerPolygonHeimdall(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)

	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.PolygonHeimdallChainID, 15, "heimdall",
		blkToWait, nil, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("heimdall chain %d already registered", config.DefConfig.PolygonHeimdallChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("heimdall chain %d already requested", config.DefConfig.PolygonHeimdallChainID)
			return true
		}
		panic(fmt.Errorf("registerOK failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register heimdall chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func registerOK(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {

	blkToWait := uint64(1)
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.OkEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("registerMSC, failed to decode eccd '%s' : %v", config.DefConfig.OkEccd, err))
	}

	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.OkChainID, 12, "ok",
		blkToWait, eccd, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("ok chain %d already registered", config.DefConfig.OkChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("ok chain %d already requested", config.DefConfig.OkChainID)
			return true
		}
		panic(fmt.Errorf("registerOK failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register ok chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterXdai(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {

	blkToWait := uint64(1)

	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.XdaiEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterO3, failed to decode eccd '%s' : %v", config.DefConfig.XdaiEccd, err))
	}

	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.XdaiChainID, config.DefConfig.Router, config.DefConfig.Name,
		blkToWait, eccd, nil, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("chain %d already registered", config.DefConfig.XdaiChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("chain %d already requested", config.DefConfig.XdaiChainID)
			return true
		}
		panic(fmt.Errorf("RegisterArb failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterArb(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {

	blkToWait := uint64(1)

	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.ArbEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterO3, failed to decode eccd '%s' : %v", config.DefConfig.ArbEccd, err))
	}

	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.ArbChainID, 0, "arb",
		blkToWait, eccd, nil, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("arb chain %d already registered", config.DefConfig.ArbChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("arb chain %d already requested", config.DefConfig.ArbChainID)
			return true
		}
		panic(fmt.Errorf("RegisterArb failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register arb chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterO3(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	tool := eth.NewEthTools(config.DefConfig.O3URL)
	chainID, err := tool.GetChainID()
	if err != nil {
		panic(err)
	}

	blkToWait := uint64(15)
	extra := msc.ExtraInfo{
		ChainID: chainID,
		Period:  3,
		Epoch:   200,
	}

	extraBytes, _ := json.Marshal(extra)

	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.O3Eccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterO3, failed to decode eccd '%s' : %v", config.DefConfig.O3Eccd, err))
	}

	fmt.Println("config.DefConfig.O3ChainID", config.DefConfig.O3ChainID, "extraBytes", string(extraBytes), "O3Eccd", config.DefConfig.O3Eccd)
	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.O3ChainID, 10, "o3",
		blkToWait, eccd, extraBytes, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("o3 chain %d already registered", config.DefConfig.O3ChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("o3 chain %d already requested", config.DefConfig.O3ChainID)
			return true
		}
		panic(fmt.Errorf("RegisterO3 failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register o3 chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterHeco(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	tool := eth.NewEthTools(config.DefConfig.HecoURL)
	chainID, err := tool.GetChainID()
	if err != nil {
		panic(err)
	}

	blkToWait := uint64(21)
	extra := heco.ExtraInfo{
		ChainID: chainID,
		Period:  3,
	}

	extraBytes, _ := json.Marshal(extra)
	log.Errorf("Heco Extra: %+v", extra)
	log.Errorf("Heco Extra Bytes: %x", extraBytes)
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.HecoEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterHeco, failed to decode eccd '%s' : %v", config.DefConfig.HecoEccd, err))
	}

	fmt.Println("config.DefConfig.HecoChainID: ", config.DefConfig.HecoChainID, "extraBytes: ", string(extraBytes), "HecoEccd: ", config.DefConfig.HecoEccd)
	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.HecoChainID, config.DefConfig.Router, config.DefConfig.Name,
		blkToWait, eccd, extraBytes, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("heco chain %d already registered", config.DefConfig.HecoChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("heco chain %d already requested", config.DefConfig.HecoChainID)
			return true
		}
		panic(fmt.Errorf("RegisterHeco failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register heco chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterBSC(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	tool := eth.NewEthTools(config.DefConfig.BSCURL)
	chainID, err := tool.GetChainID()
	if err != nil {
		panic(err)
	}

	blkToWait := uint64(15)
	extra := bsc.ExtraInfo{
		ChainID: chainID,
	}

	extraBytes, _ := json.Marshal(extra)

	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.BscEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterBSC, failed to decode eccd '%s' : %v", config.DefConfig.BscEccd, err))
	}

	fmt.Println("config.DefConfig.BSCChainID", config.DefConfig.BscChainID, "extraBytes", string(extraBytes), "BscEccd", config.DefConfig.BscEccd)
	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.BscChainID, config.DefConfig.Router, config.DefConfig.Name,
		blkToWait, eccd, extraBytes, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("bsc chain %d already registered", config.DefConfig.BscChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("bsc chain %d already requested", config.DefConfig.BscChainID)
			return true
		}
		panic(fmt.Errorf("RegisterBSC failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register bsc chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func RegisterZIL(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {

	type ExtraInfo struct {
		NumOfGuardList int
	}

	numOfGuardList := 0
	zilSdk := provider.NewProvider(config.DefConfig.ZilURL)
	dsComm, err := zilSdk.GetCurrentDSComm()
	if err != nil {
		panic(fmt.Errorf("RegisterZIL failed: %s", err.Error()))
	}
	numOfGuardList = dsComm.NumOfDSGuard

	fmt.Println("register zil")
	blkToWait := uint64(15)

	extra := ExtraInfo{NumOfGuardList: numOfGuardList}
	extraBytes, _ := json.Marshal(extra)

	// todo cross chain manger or its proxy
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.ZilEccdImpl, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("RegisterZIL, failed to decode eccd '%s' : %v", config.DefConfig.ZilEccdImpl, err))
	}

	txhash, err := poly.Native.Scm.RegisterSideChainExt(acc.Address, config.DefConfig.ZilChainID, 17, "zil",
		blkToWait, eccd, extraBytes, acc)

	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("zil chain %d already registered", config.DefConfig.ZilChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("zil chain %d already requested", config.DefConfig.ZilChainID)
			return true
		}
		panic(fmt.Errorf("RegisterZIL failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register zil chain: ( txhash: %s )", txhash.ToHexString())
	return true
}

func RegisterOntChain(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	txhash, err := poly.Native.Scm.RegisterSideChain(acc.Address, config.DefConfig.OntChainID, 3, "ont",
		1, []byte{}, acc)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			log.Infof("ont chain %d already registered", config.DefConfig.OntChainID)
			return false
		}
		if strings.Contains(err.Error(), "already requested") {
			log.Infof("ont chain %d already requested", config.DefConfig.OntChainID)
			return true
		}
		panic(fmt.Errorf("RegisterOntChain failed: %v", err))
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register ont chain: ( txhash: %s )", txhash.ToHexString())

	return true
}

func UpdateBtc(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	var tyNet utils.BtcNetType
	switch config.BtcNet.Name {
	case "testnet3":
		blkToWait = 6
		tyNet = utils.TyTestnet3
	case "mainnet":
		blkToWait = 6
		tyNet = utils.TyMainnet
	case "regtest":
		tyNet = utils.TyRegtest
	case "simnet":
		tyNet = utils.TySimnet
	}

	rawTy := make([]byte, 8)
	binary.LittleEndian.PutUint64(rawTy, uint64(tyNet))

	if err := updateSideChain(poly, acc, config.DefConfig.BtcChainID, 1, blkToWait, "btc", rawTy); err != nil {
		log.Errorf("failed to update btc: %v", err)
		return false
	}

	return true
}

func UpdateEth(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	if config.BtcNet.Name == "testnet3" {
		blkToWait = 12
	}
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.Eccd, "0x", "", 1))
	if err != nil {
		log.Errorf("failed to decode eccd: %v", err)
		return false
	}
	if err = updateSideChain(poly, acc, config.DefConfig.EthChainID, 2, blkToWait, "eth", eccd); err != nil {
		log.Errorf("failed to update eth: %v", err)
		return false
	}
	return true
}

func UpdateNeo(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	eccd, err := common2.AddressFromHexString(strings.TrimPrefix(config.DefConfig.NeoCCMC, "0x"))
	if err != nil {
		log.Errorf("failed to decode eccd: %v", err)
		return false
	}
	if err = updateSideChain(poly, acc, config.DefConfig.NeoChainID, 4, blkToWait, "NEO", eccd[:]); err != nil {
		log.Errorf("failed to update neo: %v", err)
		return false
	}
	return true
}

func UpdateZil(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.ZilEccdImpl, "0x", "", 1))
	if err != nil {
		log.Errorf("failed to decode eccd: %v", err)
		return false
	}

	type ExtraInfo struct {
		NumOfGuardList int
	}
	extra := ExtraInfo{NumOfGuardList: 10}
	extraBytes, _ := json.Marshal(extra)

	if err = updateSideChainExt(poly, acc, config.DefConfig.ZilChainID, 9, blkToWait, "zil", eccd[:], extraBytes); err != nil {
		log.Errorf("failed to update neo: %v", err)
		return false
	}
	return true
}

func UpdateNeo3(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	neo3Ccmc := helper3.HexToBytes(config.DefConfig.Neo3CCMC)
	if len(neo3Ccmc) != 4 { //uint32
		log.Errorf("incorrect Neo3CCMC length")
		return false
	}
	if err := updateSideChainExt(poly, acc, config.DefConfig.Neo3ChainID, 14, blkToWait, "NEO3", neo3Ccmc[:], helper3.UInt32ToBytes(config.DefConfig.Neo3Magic)); err != nil {
		log.Errorf("failed to update neo3: %v", err)
		return false
	}
	return true
}

func UpdateOK(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	blkToWait := uint64(1)
	eccd, err := hex.DecodeString(strings.Replace(config.DefConfig.OkEccd, "0x", "", 1))
	if err != nil {
		panic(fmt.Errorf("UpdateOK, failed to decode eccd '%s' : %v", config.DefConfig.OkEccd, err))
	}

	if err := updateSideChain(poly, acc, config.DefConfig.OkChainID, 12, blkToWait, "okex", eccd); err != nil {
		log.Errorf("failed to update ok: %v", err)
		return false
	}
	return true
}

func updateSideChain(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account, chainId, router, blkToWait uint64, name string,
	ccmc []byte) error {
	txhash, err := poly.Native.Scm.UpdateSideChain(acc.Address, chainId, router, name, blkToWait, ccmc, acc)
	if err != nil {
		return err
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to update %s: ( txhash: %s )", name, txhash.ToHexString())
	return nil
}

func updateSideChainExt(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account, chainId, router, blkToWait uint64, name string,
	ccmc []byte, extra []byte) error {
	txhash, err := poly.Native.Scm.UpdateSideChainExt(acc.Address, chainId, router, name, blkToWait, ccmc, extra, acc)
	if err != nil {
		return err
	}

	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to update %s: ( txhash: %s )", name, txhash.ToHexString())
	return nil
}

func ApproveUpdateChain(id uint64, poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)

	for i, a := range accArr {
		txhash, err = poly.Native.Scm.ApproveUpdateSideChain(id, a)
		if err != nil {
			panic(fmt.Errorf("No%d ApproveUpdateChain failed: %v", i, err))
		}
		log.Infof("No%d: successful to approve update chain: ( acc: %s, txhash: %s, chain-id: %d )",
			i, a.Address.ToHexString(), txhash.ToHexString(), id)
	}
	testcase.WaitPolyTx(txhash, poly)
}

func RegisterCandidate(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) bool {
	txHash, err := poly.Native.Nm.RegisterCandidate(vconfig.PubkeyID(acc.GetPublicKey()), acc)
	if err != nil {
		if strings.Contains(err.Error(), "already") {
			log.Warnf("candidate %s already registered: %v", acc.Address.ToBase58(), err)
			return true
		}
		log.Errorf("sendTransaction error: %v", err)
	}
	testcase.WaitPolyTx(txHash, poly)
	log.Infof("successful to register candidate: ( candidate: %s, txhash: %s )",
		acc.Address.ToHexString(), txHash.ToHexString())

	return true
}

func ApproveCandidate(acc *poly_go_sdk.Account, poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, a := range accArr {
		txhash, err = poly.Native.Nm.ApproveCandidate(vconfig.PubkeyID(acc.GetPublicKey()), a)
		if err != nil {
			panic(fmt.Errorf("no%d sendTransaction error: %v", i, err))
		}
		log.Infof("No%d: successful to approve candidate: ( acc: %s, txhash: %s, candidate: %s )",
			i, a.Address.ToHexString(), txhash.ToHexString(), acc.Address.ToHexString())
	}
	testcase.WaitPolyTx(txhash, poly)
}

func BlackPolyNode(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, v := range accArr {
		txhash, err = poly.Native.Nm.BlackNode([]string{vconfig.PubkeyID(acc.PublicKey)}, v)
		if err != nil {
			panic(fmt.Errorf("no%d - failed to black node %s: %v", i, acc.Address.ToHexString(), err))
		}
		log.Infof("No%d: successful to black node: ( acc: %s, txhash: %s, node_to_black: %s )",
			i, v.Address.ToHexString(), txhash.ToHexString(), acc.Address.ToHexString())
	}
	testcase.WaitPolyTx(txhash, poly)
}

func WhitePolyNode(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, v := range accArr {
		txhash, err = poly.Native.Nm.WhiteNode(vconfig.PubkeyID(acc.PublicKey), v)
		if err != nil {
			panic(fmt.Errorf("no%d - failed to white node %s: %v", i, acc.Address.ToHexString(), err))
		}
		log.Infof("No%d: successful to white node: ( acc: %s, txhash: %s, node_to_white: %s )",
			i, v.Address.ToHexString(), txhash.ToHexString(), acc.Address.ToHexString())
	}
	testcase.WaitPolyTx(txhash, poly)
}

func CommitPolyDpos(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	txhash, err := poly.Native.Nm.CommitDpos(accArr)
	if err != nil {
		panic(err)
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to commit dpos on Poly: txhash: %s", txhash.ToHexString())
}

func CommitOntDpos() {
	ontCli := ontology_go_sdk.NewOntologySdk()
	ontCli.NewRpcClient().SetAddress(config.DefConfig.OntJsonRpcAddress)
	tx, err := ontCli.Native.NewNativeInvokeTransaction(config.DefConfig.GasPrice, config.DefConfig.GasLimit, 0, ontology_go_sdk.GOVERNANCE_CONTRACT_ADDRESS, governance.COMMIT_DPOS, nil)
	if err != nil {
		panic(fmt.Errorf("NewNativeInvokeTransaction error: %+v\n", err))
	}
	ow := strings.Split(oWalletFiles, ",")
	op := strings.Split(oPwds, ",")
	oAccArr := make([]*ontology_go_sdk.Account, len(ow))
	pks := make([]keypair.PublicKey, len(ow))
	for i, v := range ow {
		oAccArr[i], err = ont.GetOntAccByPwd(v, op[i])
		if err != nil {
			panic(fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, ow[i], op[i]))
		}
		pks[i] = oAccArr[i].PublicKey
	}
	for _, signer := range oAccArr {
		err = ontCli.MultiSignToTransaction(tx, uint16((5*len(pks)+6)/7), pks, signer)
		if err != nil {
			panic(fmt.Errorf("multi sign failed, err: %s\n", err))
		}
	}
	txHash, err := ontCli.SendTransaction(tx)
	if err != nil {
		panic(fmt.Errorf("sendTransaction error: %+v\n", err))
	}
	ont.WaitOntTx(txHash, ontCli)
	log.Infof("successful to commit dpos on Ontology: txhash: %s", txHash.ToHexString())
}

func QuitNode(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) {
	txhash, err := poly.Native.Nm.QuitNode(vconfig.PubkeyID(acc.PublicKey), acc)
	if err != nil {
		panic(fmt.Errorf("failed to quit %s: %v", acc.Address.ToBase58(), err))
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to quit node %s on Poly: txhash: %s", acc.Address.ToBase58(), txhash.ToHexString())
}

func GetPolyConsensusInfo(poly *poly_go_sdk.PolySdk) {
	storeBs, err := poly.GetStorage(utils.NodeManagerContractAddress.ToHexString(), []byte(node_manager.GOVERNANCE_VIEW))
	if err != nil {
		panic(err)
	}
	source := common.NewZeroCopySource(storeBs)
	gv := new(node_manager.GovernanceView)
	if err := gv.Deserialization(source); err != nil {
		panic(err)
	}

	raw, err := poly.GetStorage(utils.NodeManagerContractAddress.ToHexString(),
		append([]byte(node_manager.PEER_POOL), utils.GetUint32Bytes(gv.View)...))
	if err != nil {
		panic(err)
	}
	m := &node_manager.PeerPoolMap{
		PeerPoolMap: make(map[string]*node_manager.PeerPoolItem),
	}
	if err := m.Deserialization(common.NewZeroCopySource(raw)); err != nil {
		panic(err)
	}
	str := ""
	for _, v := range m.PeerPoolMap {
		str += fmt.Sprintf("[ index: %d, address: %s, pubk: %s, status: %d ]\n",
			v.Index, v.Address.ToBase58(), v.PeerPubkey, v.Status)
	}

	log.Infof("get consensus info of poly: { view: %d, len_nodes: %d, info: \n%s }", gv.View, len(m.PeerPoolMap), str)
}

func RegisterRelayer(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account, signer *poly_go_sdk.Account) uint64 {
	txhash, err := poly.Native.Rm.RegisterRelayer([]common.Address{acc.Address}, signer)
	if err != nil {
		panic(err)
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register a relayer %s, txhash is %s", acc.Address.ToBase58(), txhash.ToHexString())
	event, err := poly.GetSmartContractEvent(txhash.ToHexString())
	if err != nil {
		panic(err)
	}
	var id uint64
	for _, e := range event.Notify {
		states := e.States.([]interface{})
		if states[0].(string) == "putRelayerApply" {
			id = uint64(states[1].(float64))
		}
	}
	return id
}

func ApproveRelayer(poly *poly_go_sdk.PolySdk, id uint64, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, v := range accArr {
		txhash, err = poly.Native.Rm.ApproveRegisterRelayer(id, v)
		if err != nil {
			panic(fmt.Errorf("no%d - failed to approve %d: %v", i, id, err))
		}
		log.Infof("No%d: successful to approve relayer id %d: ( acc: %s, txhash: %s )",
			i, id, v.Address.ToHexString(), txhash.ToHexString())
	}
	testcase.WaitPolyTx(txhash, poly)
}

func RemoveRelayer(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) uint64 {
	txhash, err := poly.Native.Rm.RemoveRelayer([]common.Address{acc.Address}, acc)
	if err != nil {
		panic(err)
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to remove a relayer %s, txhash is %s", acc.Address.ToBase58(), txhash.ToHexString())
	event, err := poly.GetSmartContractEvent(txhash.ToHexString())
	if err != nil {
		panic(err)
	}
	var id uint64
	for _, e := range event.Notify {
		states := e.States.([]interface{})
		if states[0].(string) == "putRelayerRemove" {
			id = uint64(states[1].(float64))
		}
	}
	return id
}

func ApproveRemoveRelayer(poly *poly_go_sdk.PolySdk, id uint64, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, v := range accArr {
		txhash, err = poly.Native.Rm.ApproveRemoveRelayer(id, v)
		if err != nil {
			panic(fmt.Errorf("no%d - failed to approve %d: %v", i, id, err))
		}
		log.Infof("No%d: successful to approve remove relayer id %d: ( acc: %s, txhash: %s )",
			i, id, v.Address.ToHexString(), txhash.ToHexString())
	}
	testcase.WaitPolyTx(txhash, poly)
}

func GetRelayer(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) {
	raw, err := poly.GetStorage(utils.RelayerManagerContractAddress.ToHexString(),
		append([]byte(relayer_manager.RELAYER), acc.Address[:]...))
	if err != nil {
		panic(err)
	}
	if len(raw) == 0 {
		log.Infof("no this relayer %s", acc.Address.ToBase58())
		return
	}
	addr, err := common.AddressParseFromBytes(raw)
	if err != nil {
		panic(err)
	}
	log.Infof("get relayer success: %s", addr.ToBase58())
}

func RegisterStateValidator(poly *poly_go_sdk.PolySdk, neo3PubKeys []string, signer *poly_go_sdk.Account) uint64 {
	txhash, err := poly.Native.Sm.RegisterStateValidator(neo3PubKeys, signer)
	if err != nil {
		panic(err)
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to register neo3 state validators, txHash is %s", txhash.ToHexString())
	event, err := poly.GetSmartContractEvent(txhash.ToHexString())
	if err != nil {
		panic(err)
	}
	var id uint64
	for _, e := range event.Notify {
		states := e.States.([]interface{})
		if states[0].(string) == "putStateValidatorApply" {
			id = uint64(states[1].(float64))
		}
	}
	return id
}

func ApproveRegisterStateValidator(poly *poly_go_sdk.PolySdk, id uint64, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, v := range accArr {
		txhash, err = poly.Native.Sm.ApproveRegisterStateValidator(id, v)
		if err != nil {
			panic(fmt.Errorf("no%d - failed to approve %d: %v", i, id, err))
		}
		log.Infof("No%d: successful to approve register state validators id %d: ( acc: %s, txHash: %s )",
			i, id, v.Address.ToHexString(), txhash.ToHexString())
		testcase.WaitPolyTx(txhash, poly)
	}
}

func RemoveStateValidator(poly *poly_go_sdk.PolySdk, neo3PubKeys []string, signer *poly_go_sdk.Account) uint64 {
	txhash, err := poly.Native.Sm.RemoveStateValidator(neo3PubKeys, signer)
	if err != nil {
		panic(err)
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to remove neo3 state validators, txHash is %s", txhash.ToHexString())
	event, err := poly.GetSmartContractEvent(txhash.ToHexString())
	if err != nil {
		panic(err)
	}
	var id uint64
	for _, e := range event.Notify {
		states := e.States.([]interface{})
		if states[0].(string) == "putStateValidatorRemove" {
			id = uint64(states[1].(float64))
		}
	}
	return id
}

func ApproveRemoveStateValidator(poly *poly_go_sdk.PolySdk, id uint64, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, v := range accArr {
		txhash, err = poly.Native.Sm.ApproveRemoveStateValidator(id, v)
		if err != nil {
			panic(fmt.Errorf("no%d - failed to approve %d: %v", i, id, err))
		}
		log.Infof("No%d: successful to approve remove state validators id %d: ( acc: %s, txHash: %s )",
			i, id, v.Address.ToHexString(), txhash.ToHexString())
	}
	testcase.WaitPolyTx(txhash, poly)
}

func GetStateValidator(poly *poly_go_sdk.PolySdk) {
	raw, err := poly.GetStorage(utils.Neo3StateManagerContractAddress.ToHexString(), append([]byte(neo3_state_manager.STATE_VALIDATOR)))
	if err != nil {
		panic(err)
	}
	if len(raw) == 0 {
		log.Infof("no state validators found")
		return
	}
	svBytes, err := states.GetValueFromRawStorageItem(raw)
	if err != nil {
		panic(err)
	}
	svs, err := neo3_state_manager.DeserializeStringArray(svBytes)
	if err != nil {
		panic(err)
	}
	for _, v := range svs {
		log.Infof("neo3 state validator pubKey: %s", v)
	}
}

func QuitSideChain(poly *poly_go_sdk.PolySdk, id uint64, acc *poly_go_sdk.Account) {
	txhash, err := poly.Native.Scm.QuitSideChain(id, acc)
	if err != nil {
		panic(fmt.Errorf("failed to quit %s: %v", acc.Address.ToBase58(), err))
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("successful to quit side chain %s on Poly: txhash: %s", acc.Address.ToBase58(), txhash.ToHexString())
}

func ApproveQuitSideChain(poly *poly_go_sdk.PolySdk, id uint64, accArr []*poly_go_sdk.Account) {
	var (
		txhash common.Uint256
		err    error
	)
	for i, v := range accArr {
		txhash, err = poly.Native.Scm.ApproveQuitSideChain(id, v)
		if err != nil {
			panic(fmt.Errorf("no%d - failed to approve %d: %v", i, id, err))
		}
		log.Infof("No%d: successful to approve quit side chain %d: ( acc: %s, txhash: %s )",
			i, id, v.Address.ToHexString(), txhash.ToHexString())
	}
	testcase.WaitPolyTx(txhash, poly)
}

func GetSideChain(poly *poly_go_sdk.PolySdk, id uint64) {
	store, err := poly.GetStorage(utils.SideChainManagerContractAddress.ToHexString(),
		append([]byte(side_chain_manager.SIDE_CHAIN), utils.GetUint64Bytes(id)...))
	if err != nil {
		panic(err)
	}
	if store == nil {
		log.Infof("no this %d side chain found", id)
		return
	}
	sideChain := new(side_chain_manager.SideChain)
	err = sideChain.Deserialization(common.NewZeroCopySource(store))
	if err != nil {
		panic(err)
	}
	log.Infof("side chain %d, name: %s, addr: %s", id, sideChain.Name, sideChain.Address.ToBase58())
}

func UpdatePolyConfig(poly *poly_go_sdk.PolySdk, blockMsgDelay, hashMsgDelay, peerHandshakeTimeout,
	maxBlockChangeView uint32, accArr []*poly_go_sdk.Account) {
	txhash, err := poly.Native.Nm.UpdateConfig(blockMsgDelay, hashMsgDelay, peerHandshakeTimeout, maxBlockChangeView, accArr)
	if err != nil {
		panic(err)
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("update poly config: "+
		"(blockMsgDelay: %d, hashMsgDelay: %d, peerHandshakeTimeout: %d, maxBlockChangeView: %d)",
		blockMsgDelay, hashMsgDelay, peerHandshakeTimeout, maxBlockChangeView)
}

func GetPolyConfig(poly *poly_go_sdk.PolySdk) {
	raw, err := poly.GetStorage(utils.NodeManagerContractAddress.ToHexString(), []byte(node_manager.VBFT_CONFIG))
	if err != nil {
		panic(err)
	}
	conf := &node_manager.Configuration{}
	if err = conf.Deserialization(common.NewZeroCopySource(raw)); err != nil {
		panic(err)
	}
	log.Infof("poly config: (blockMsgDelay: %d, hashMsgDelay: %d, peerHandshakeTimeout: %d, maxBlockChangeView: %d)",
		conf.BlockMsgDelay, conf.HashMsgDelay, conf.PeerHandshakeTimeout, conf.MaxBlockChangeView)
}

func UnRegisterPolyCandidate(poly *poly_go_sdk.PolySdk, acc *poly_go_sdk.Account) {
	txhash, err := poly.Native.Nm.UnRegisterCandidate(vconfig.PubkeyID(acc.PublicKey), acc)
	if err != nil {
		panic(err)
	}
	testcase.WaitPolyTx(txhash, poly)
	log.Infof("unregister %s success: txhash: %s", acc.Address.ToBase58(), txhash.ToHexString())
}

func CosmosCreateValidator(keyFile, stateFile string, amt int64) {
	invoker, err := cosmos2.NewCosmosInvoker()
	if err != nil {
		panic(err)
	}

	dc, err := types3.ParseDecCoins(config.DefConfig.CMGasPrice)
	if err != nil {
		panic(err)
	}
	demon := dc.GetDenomByIndex(0)
	res, err := invoker.CreateValidator(keyFile, stateFile, demon, amt)
	if err != nil {
		panic(err)
	}
	log.Infof("CosmosCreateValidator, create a new validator %s with %d %s: txhash %s",
		invoker.Acc.Acc.String(), amt, demon, res.Hash.String())
}

func CosmosDelegateToVal(amt int64) {
	invoker, err := cosmos2.NewCosmosInvoker()
	if err != nil {
		panic(err)
	}

	dc, err := types3.ParseDecCoins(config.DefConfig.CMGasPrice)
	if err != nil {
		panic(err)
	}
	demon := dc.GetDenomByIndex(0)
	res, err := invoker.DelegateValidator(demon, amt)
	if err != nil {
		panic(err)
	}
	log.Infof("CosmosDelegateToVal, delegate %d %s to validator %s: txhash %s",
		amt, demon, types3.ValAddress(invoker.Acc.Acc).String(), res.Hash.String())
}
