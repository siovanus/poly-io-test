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
	"flag"
	"fmt"

	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/polynetwork/poly-io-test/chains/eth"
	"github.com/polynetwork/poly-io-test/config"
)

var (
	fnEth        string
	ethConfFile  string
	eccmRedeploy int
)

func init() {
	flag.StringVar(&fnEth, "func", "deploy", "choose function to run: deploy or setup")
	flag.StringVar(&ethConfFile, "conf", "./config.json", "config file path")
	flag.IntVar(&eccmRedeploy, "redeploy_eccm", 1, "redeploy eccd, eccm and eccmp or not")
	flag.Parse()
}

func main() {
	err := config.DefConfig.Init(ethConfFile)
	if err != nil {
		panic(err)
	}

	switch fnEth {
	case "deploy":
		DeployETHSmartContract()
	case "setup":
		SetUpEthContracts()
	}
}

func DeployETHSmartContract() {
	invoker := eth.NewEInvoker(config.DefConfig.XdaiChainID)
	var (
		eccdAddr  common2.Address
		eccmAddr  common2.Address
		eccmpAddr common2.Address
		err       error
	)
	if eccmRedeploy == 1 {
		eccdAddr, _, err = invoker.DeployEthChainDataContract()
		if err != nil {
			panic(err)
		}

		eccmAddr, _, err = invoker.DeployECCMContract(eccdAddr.Hex())
		if err != nil {
			panic(err)
		}
		eccmpAddr, _, err = invoker.DeployECCMPContract(eccmAddr.Hex())
		if err != nil {
			panic(err)
		}
		_, err = invoker.TransferOwnershipForECCD(eccdAddr.Hex(), eccmAddr.Hex())
		if err != nil {
			panic(err)
		}
		_, err = invoker.TransferOwnershipForECCM(eccmAddr.Hex(), eccmpAddr.Hex())
		if err != nil {
			panic(err)
		}
	} else {
		eccdAddr = common2.HexToAddress(config.DefConfig.XdaiEccd)
		eccmAddr = common2.HexToAddress(config.DefConfig.XdaiEccm)
		eccmpAddr = common2.HexToAddress(config.DefConfig.XdaiEccmp)
	}

	lockProxyAddr, _, err := invoker.DeployLockProxyContract(eccmpAddr)
	if err != nil {
		panic(err)
	}

	fmt.Println("=============================ETH info=============================")
	fmt.Println("eccd address:", eccdAddr.Hex())
	fmt.Println("eccm address:", eccmAddr.Hex())
	fmt.Println("eccmp address:", eccmpAddr.Hex())
	fmt.Println("lock proxy address: ", lockProxyAddr.Hex())
	fmt.Println("==================================================================")

	config.DefConfig.XdaiEccm = eccmAddr.Hex()
	config.DefConfig.XdaiEccmp = eccmpAddr.Hex()
	config.DefConfig.XdaiLockProxy = lockProxyAddr.Hex()
	config.DefConfig.XdaiEccd = eccdAddr.Hex()

	if err := config.DefConfig.Save(ethConfFile); err != nil {
		panic(fmt.Errorf("failed to save config, you better save it youself: %v", err))
	}
}

func SetupNative(ethInvoker *eth.EInvoker) {
	ethNativeAddr := "0x0000000000000000000000000000000000000000"

	tx, err := ethInvoker.BindAssetHash(config.DefConfig.XdaiLockProxy, ethNativeAddr, ethNativeAddr, config.DefConfig.XdaiChainID, 0)
	if err != nil {
		panic(fmt.Errorf("BindAssetHash, failed to bind asset hash: %v", err))
	}
	hash := tx.Hash()
	fmt.Printf("binding matic of polygon: ( txhash: %s )\n", hash.String())

	if config.DefConfig.XdaiLockProxy != "" {
		_, contract, err := ethInvoker.MakeLockProxy(config.DefConfig.XdaiLockProxy)
		if err != nil {
			panic(fmt.Errorf("failed to MakeLockProxy: %v", err))
		}
		auth, err := ethInvoker.MakeSmartContractAuth()
		if err != nil {
			panic(fmt.Errorf("failed to get auth: %v", err))
		}
		other := common2.HexToAddress(config.DefConfig.XdaiLockProxy)
		tx, err := contract.BindProxyHash(auth, config.DefConfig.XdaiChainID, other[:])
		if err != nil {
			panic(fmt.Errorf("failed to bind proxy: %v", err))
		}
		hash := tx.Hash()
		fmt.Printf("binding proxy: ( txhash: %s )\n", hash.String())
	}
}

func SetUpEthContracts() {
	invoker := eth.NewEInvoker(config.DefConfig.XdaiChainID)
	SetupNative(invoker)
}
