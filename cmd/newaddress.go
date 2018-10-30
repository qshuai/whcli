// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"strings"

	"github.com/bcext/cashutil"
	"github.com/bcext/gcash/btcec"
	"github.com/bcext/gcash/chaincfg"
	"github.com/qshuai/tcolor"
	"github.com/spf13/cobra"
)

// newaddressCmd represents the newaddress command
var newaddressCmd = &cobra.Command{
	Use:   "newaddress mainnet/testnet/regtest",
	Args:  cobra.MinimumNArgs(1),
	Short: "Generate a new address based on system random seed",
	Long: `Generate a safe bitcoin cash address(including base58 and bech32 encoded
format).

Caution: please save the private key and the corresponding address by yourself. 
whcli will not touch your private key never. So whcli does not have the obligation
for reserve your own keys.`,
	Run: func(cmd *cobra.Command, args []string) {
		getNewAddress(args)
	},
}

var net = map[string]*chaincfg.Params{
	"mainnet": &chaincfg.MainNetParams,
	"testnet": &chaincfg.TestNet3Params,
	"regtest": &chaincfg.RegressionNetParams,
}

func getNewAddress(args []string) {
	if len(args) != 1 {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Usage: addr mainnet/testnet/regtest"))
		return
	}

	var n *chaincfg.Params
	n, ok := net[strings.ToLower(args[0])]
	if !ok {
		fmt.Println(tcolor.WithColor(tcolor.Red, args[0]+" not existed, should select from mainnet/testnet/regtest"))
		return
	}

	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Create private key failed: "+err.Error()))
		return
	}

	wif, err := cashutil.NewWIF(priv, n, true)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Convert wif format private key failed: "+err.Error()))
		return
	}

	pubKey := priv.PubKey()
	pubKeyHash := cashutil.Hash160(pubKey.SerializeCompressed())
	addr, err := cashutil.NewAddressPubKeyHash(pubKeyHash, n)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Generate a new bitcoin-cash address failed: "+err.Error()))
		return
	}

	fmt.Println("privkey key:           ", tcolor.WithColor(tcolor.Green, wif.String()))
	fmt.Println("base58 encoded address:", tcolor.WithColor(tcolor.Green, addr.EncodeAddress(false)))
	fmt.Println("bech32 encoded address:", tcolor.WithColor(tcolor.Green, addr.EncodeAddress(true)))
}

func init() {
	rootCmd.AddCommand(newaddressCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// newaddressCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// newaddressCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
