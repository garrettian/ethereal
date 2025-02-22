// Copyright © 2017-2019 Weald Technology Trading
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
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/cobra"
	"github.com/wealdtech/ethereal/cli"
	"github.com/wealdtech/ethereal/util/txdata"
	ens "github.com/wealdtech/go-ens/v2"
	string2eth "github.com/wealdtech/go-string2eth"
)

var transactionInfoRaw bool
var transactionInfoJSON bool
var transactionInfoSignatures string

// transactionInfoCmd represents the transaction info command
var transactionInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Obtain information about a transaction",
	Long: `Obtain information about a transaction.  For example:

    ethereal transaction info --transaction=0x5FfC014343cd971B7eb70732021E26C35B744cc4

In quiet mode this will return 0 if the transaction exists, otherwise 1.`,
	Run: func(cmd *cobra.Command, args []string) {
		cli.Assert(transactionStr != "", quiet, "--transaction is required")
		var txHash common.Hash
		var pending bool
		var tx *types.Transaction
		if len(transactionStr) > 66 {
			// Assume input is a raw transaction
			data, err := hex.DecodeString(strings.TrimPrefix(transactionStr, "0x"))
			cli.ErrCheck(err, quiet, "Failed to decode data")
			tx = &types.Transaction{}
			stream := rlp.NewStream(bytes.NewReader(data), 0)
			err = tx.DecodeRLP(stream)
			cli.ErrCheck(err, quiet, "Failed to decode raw transaction")
			txHash = tx.Hash()
		} else {
			// Assume input is a transaction ID
			txHash = common.HexToHash(transactionStr)
			ctx, cancel := localContext()
			defer cancel()
			var err error
			tx, pending, err = client.TransactionByHash(ctx, txHash)
			cli.ErrCheck(err, quiet, fmt.Sprintf("Failed to obtain transaction %s", txHash.Hex()))
		}

		if quiet {
			os.Exit(_exit_success)
		}

		if transactionInfoRaw {
			buf := new(bytes.Buffer)
			tx.EncodeRLP(buf)
			fmt.Printf("0x%s\n", hex.EncodeToString(buf.Bytes()))
			os.Exit(_exit_success)
		}

		if transactionInfoJSON {
			json, err := tx.MarshalJSON()
			cli.ErrCheck(err, quiet, fmt.Sprintf("Failed to obtain JSON for transaction %s", txHash.Hex()))
			fmt.Printf("%s\n", string(json))
			os.Exit(_exit_success)
		}

		txdata.InitFunctionMap()
		if transactionInfoSignatures != "" {
			for _, signature := range strings.Split(transactionInfoSignatures, ";") {
				txdata.AddFunctionSignature(signature)
			}
		}

		var receipt *types.Receipt
		if pending {
			if tx.To() == nil {
				fmt.Printf("Type:\t\t\tPending contract creation\n")
			} else {
				fmt.Printf("Type:\t\t\tPending transaction\n")
			}
		} else {
			if tx.To() == nil {
				fmt.Printf("Type:\t\t\tMined contract creation\n")
			} else {
				fmt.Printf("Type:\t\t\tMined transaction\n")
			}
			ctx, cancel := localContext()
			defer cancel()
			receipt, err = client.TransactionReceipt(ctx, txHash)
			if receipt != nil {
				if receipt.Status == 0 {
					fmt.Printf("Result:\t\t\tFailed\n")
				} else {
					fmt.Printf("Result:\t\t\tSucceeded\n")
				}
			}
		}

		if receipt != nil && len(receipt.Logs) > 0 {
			// We can obtain the block number from the log
			fmt.Printf("Block:\t\t\t%d\n", receipt.Logs[0].BlockNumber)
		}

		fromAddress, err := txFrom(tx)
		if err == nil {
			fmt.Printf("From:\t\t\t%v\n", ens.Format(client, fromAddress))
		}

		// To
		if tx.To() == nil {
			if receipt != nil {
				fmt.Printf("Contract address:\t%v\n", ens.Format(client, receipt.ContractAddress))
			}
		} else {
			fmt.Printf("To:\t\t\t%v\n", ens.Format(client, *tx.To()))
		}

		if verbose {
			fmt.Printf("Nonce:\t\t\t%v\n", tx.Nonce())
			fmt.Printf("Gas limit:\t\t%v\n", tx.Gas())
		}
		if receipt != nil {
			fmt.Printf("Gas used:\t\t%v\n", receipt.GasUsed)
		}
		fmt.Printf("Gas price:\t\t%v\n", string2eth.WeiToString(tx.GasPrice(), true))
		fmt.Printf("Value:\t\t\t%v\n", string2eth.WeiToString(tx.Value(), true))

		if tx.To() != nil && len(tx.Data()) > 0 {
			fmt.Printf("Data:\t\t\t%v\n", txdata.DataToString(client, tx.Data()))
		}

		if verbose && receipt != nil && len(receipt.Logs) > 0 {
			fmt.Printf("Logs:\n")
			for i, log := range receipt.Logs {
				fmt.Printf("\t%d:\n", i)
				fmt.Printf("\t\tFrom:\t%v\n", ens.Format(client, log.Address))
				// Try to obtain decoded log
				decoded := txdata.EventToString(client, log)
				if decoded != "" {
					fmt.Printf("\t\tEvent:\t%s\n", decoded)
				} else {
					if len(log.Topics) > 0 {
						fmt.Printf("\t\tTopics:\n")
						for j, topic := range log.Topics {
							fmt.Printf("\t\t\t%d:\t%v\n", j, topic.Hex())
						}
					}
					if len(log.Data) > 0 {
						fmt.Printf("\t\tData:\n")
						for j := 0; j*32 < len(log.Data); j++ {
							fmt.Printf("\t\t\t%d:\t0x%s\n", j, hex.EncodeToString(log.Data[j*32:(j+1)*32]))
						}
					}
				}
			}
		}
	},
}

func init() {
	transactionCmd.AddCommand(transactionInfoCmd)
	transactionFlags(transactionInfoCmd)
	transactionInfoCmd.Flags().BoolVar(&transactionInfoRaw, "raw", false, "Output the transaction as raw hex")
	transactionInfoCmd.Flags().BoolVar(&transactionInfoJSON, "json", false, "Output the transaction as json")
	transactionInfoCmd.Flags().StringVar(&transactionInfoSignatures, "signatures", "", "Semicolon-separated list of custom transaction signatures (e.g. myFunc(address,bytes32);myFunc2(bool)")
}
