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
	"math/big"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wealdtech/ethereal/cli"
	"github.com/wealdtech/ethereal/util/funcparser"
	ens "github.com/wealdtech/go-ens/v2"
	string2eth "github.com/wealdtech/go-string2eth"
)

var contractSendAmount string
var contractSendFromAddress string
var contractSendCall string
var contractSendReturns string

// contractSendCmd represents the contract call command
var contractSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a contract method",
	Long: `Send a contract method to the blockchain.  For example:

   ethereal contract send --contract=0xd26114cd6EE289AccF82350c8d8487fedB8A0C07 --abi="./erc20.abi" --from=0x5FfC014343cd971B7eb70732021E26C35B744cc4 --call="transfer(0x5FfC014343cd971B7eb70732021E26C35B744cc4, 10)" --passphrase=secret

   ethereal contract send --contract=0xd26114cd6EE289AccF82350c8d8487fedB8A0C07 --signature="transfer(address,uint256)" --from=0x5FfC014343cd971B7eb70732021E26C35B744cc4 --call="transfer(0x5FfC014343cd971B7eb70732021E26C35B744cc4, 10)" --passphrase=secret

This will return an exit status of 0 if the transaction is successfully submitted (and mined if --wait is supplied), 1 if the transaction is not successfully submitted, and 2 if the transaction is successfully submitted but not mined within the supplied time limit.`,
	Aliases: []string{"transaction", "transmit"},
	Run: func(cmd *cobra.Command, args []string) {
		cli.Assert(contractSendFromAddress != "", quiet, "--from is required")
		fromAddress, err := ens.Resolve(client, contractSendFromAddress)
		cli.ErrCheck(err, quiet, fmt.Sprintf("Failed to resolve from address %s", contractSendFromAddress))

		// We need to have 'call'
		cli.Assert(contractSendCall != "", quiet, "--call is required")

		contract := parseContract("")
		method, methodArgs, err := funcparser.ParseCall(client, contract, contractSendCall)
		cli.ErrCheck(err, quiet, "Failed to parse call")

		data, err := contract.Abi.Pack(method.Name, methodArgs...)
		cli.ErrCheck(err, quiet, "Failed to convert arguments")
		outputIf(verbose, fmt.Sprintf("Data is %x", data))

		cli.Assert(contractStr != "", quiet, "--contract is required")
		contractAddress, err := ens.Resolve(client, contractStr)
		cli.ErrCheck(err, quiet, fmt.Sprintf("Failed to resolve contract address %s", contractStr))

		amount := big.NewInt(0)
		if contractSendAmount != "" {
			amount, err = string2eth.StringToWei(contractSendAmount)
			cli.ErrCheck(err, quiet, fmt.Sprintf("Invalid amount %s", contractSendAmount))
		}

		// Create and sign the transaction
		signedTx, err := createSignedTransaction(fromAddress, &contractAddress, amount, gasLimit, data)
		cli.ErrCheck(err, quiet, "Failed to create contract method transaction")

		if offline {
			if !quiet {
				buf := new(bytes.Buffer)
				signedTx.EncodeRLP(buf)
				fmt.Printf("0x%s\n", hex.EncodeToString(buf.Bytes()))
			}
			os.Exit(_exit_success)
		}

		ctx, cancel := localContext()
		defer cancel()
		err = client.SendTransaction(ctx, signedTx)
		cli.ErrCheck(err, quiet, "Failed to send contract method transaction")

		handleSubmittedTransaction(signedTx, log.Fields{
			"group":   "contract",
			"command": "send",
		}, false)
	},
}

func init() {
	contractCmd.AddCommand(contractSendCmd)
	contractFlags(contractSendCmd)
	contractSendCmd.Flags().StringVar(&contractSendAmount, "amount", "", "Amount of Ether to send with the contract method")
	contractSendCmd.Flags().StringVar(&contractSendFromAddress, "from", "", "Address from which to call the contract function")
	contractSendCmd.Flags().StringVar(&contractSendCall, "call", "", "Contract function to call")
	contractSendCmd.Flags().StringVar(&contractSendReturns, "returns", "", "Comma-separated return types")
	addTransactionFlags(contractSendCmd, "Passphrase for the address from which to send the contract transaction")
}
