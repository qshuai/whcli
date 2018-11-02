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
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/bcext/cashutil"
	"github.com/bcext/gcash/chaincfg"
	"github.com/bcext/gcash/chaincfg/chainhash"
	"github.com/bcext/gcash/txscript"
	"github.com/bcext/gcash/wire"
	"github.com/qshuai/tcolor"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// spendUnconfirmedUtxoCmd represents the spendUnconfirmedUtxo command
var spendUnconfirmedUtxoCmd = &cobra.Command{
	Use:   "spendUnconfirmedUtxo",
	Args:  cobra.MinimumNArgs(4),
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		privKey := flag.String("privkey", "", "private key of the sender")
		to := flag.String("to", "", "the bitcoin cash address of receiver")
		hash := flag.String("hash", "", "previous tx hash")
		idx := flag.Int("idx", 0, "previous tx index")
		value := flag.Int("value", 0, "the utxo value")
		flag.Parse()

		if *privKey == "" || *to == "" || *hash == "" || *value == 0 {
			fmt.Println(tcolor.WithColor(tcolor.Red, "arguments are not enough(privkey/to/hash/value required)"))
			os.Exit(1)
		}

		h, err := chainhash.NewHashFromStr(*hash)
		if err != nil {
			fmt.Println(tcolor.WithColor(tcolor.Red, "not valid transaction hash: "+err.Error()))
			os.Exit(1)
		}
		// parse privkey
		wif, err := cashutil.DecodeWIF(*privKey)
		if err != nil {
			fmt.Println(tcolor.WithColor(tcolor.Red, "privvate key format error: "+err.Error()))
			os.Exit(1)
		}
		pubKey := wif.PrivKey.PubKey()
		pubKeyBytes := pubKey.SerializeCompressed()
		pkHash := cashutil.Hash160(pubKeyBytes)
		sender, err := cashutil.NewAddressPubKeyHash(pkHash, &chaincfg.TestNet3Params)
		if err != nil {
			fmt.Println(tcolor.WithColor(tcolor.Red, "address encode failed, please check your privkey: "+err.Error()))
			os.Exit(1)
		}

		dst, _ := cashutil.DecodeAddress(*to, &chaincfg.TestNet3Params)

		tx, err := assembleTx(h, int64(*value), uint32(*idx), sender, dst, wif, 0.000001)
		if err != nil {
			fmt.Println(tcolor.WithColor(tcolor.Red, "assemble transaction failed: "+err.Error()))
			os.Exit(1)
		}

		buf := bytes.NewBuffer(nil)
		err = tx.Serialize(buf)
		if err != nil {
			fmt.Println(tcolor.WithColor(tcolor.Red, "transaction serialize failed: "+err.Error()))
			os.Exit(1)
		}

		fmt.Println(tcolor.WithColor(tcolor.Green, hex.EncodeToString(buf.Bytes())))
	},
}

func init() {
	rootCmd.AddCommand(spendUnconfirmedUtxoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// spendUnconfirmedUtxoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// spendUnconfirmedUtxoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

const (
	defaultSignatureSize = 107
	defaultSequence      = 0xffffffff
)

func assembleTx(hash *chainhash.Hash, value int64, idx uint32, sender, receiver cashutil.Address, wif *cashutil.WIF, feerate float64) (*wire.MsgTx, error) {
	var tx wire.MsgTx
	tx.Version = 1
	tx.LockTime = 0

	tx.TxOut = make([]*wire.TxOut, 1)
	pkScript, err := txscript.PayToAddrScript(receiver)
	if err != nil {
		return nil, err
	}
	tx.TxOut[0] = &wire.TxOut{PkScript: pkScript}

	outpoint := wire.NewOutPoint(hash, idx)
	tx.TxIn = append(tx.TxIn, wire.NewTxIn(outpoint, nil))
	tx.TxIn[0].Sequence = defaultSequence

	txsize := tx.SerializeSize() + defaultSignatureSize

	fee := decimal.NewFromFloat(feerate * 1e5).Mul(decimal.New(int64(txsize), 0)).Truncate(0).IntPart()
	outValue := value - fee
	tx.TxOut[0].Value = outValue

	sourcePkScript, err := txscript.PayToAddrScript(sender)
	if err != nil {
		return nil, err
	}
	// sign the transaction
	return sign(&tx, []int64{value}, sourcePkScript, wif)
}

func sign(tx *wire.MsgTx, inputValueSlice []int64, pkScript []byte, wif *cashutil.WIF) (*wire.MsgTx, error) {
	for idx, _ := range tx.TxIn {
		sig, err := txscript.RawTxInSignature(tx, idx, pkScript, cashutil.Amount(inputValueSlice[idx]),
			txscript.SigHashAll|txscript.SigHashForkID, wif.PrivKey)
		if err != nil {
			return nil, err
		}
		sig, err = txscript.NewScriptBuilder().AddData(sig).Script()
		if err != nil {
			return nil, err
		}
		pk, err := txscript.NewScriptBuilder().AddData(wif.PrivKey.PubKey().SerializeCompressed()).Script()
		if err != nil {
			return nil, err
		}
		sig = append(sig, pk...)
		tx.TxIn[0].SignatureScript = sig

		engine, err := txscript.NewEngine(pkScript, tx, idx, txscript.StandardVerifyFlags,
			nil, nil, inputValueSlice[idx])
		if err != nil {
			return nil, err
		}

		// verify the signature
		err = engine.Execute()
		if err != nil {
			return nil, err
		}
	}

	return tx, nil
}
