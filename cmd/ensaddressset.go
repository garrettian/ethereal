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
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wealdtech/ethereal/cli"
	ens "github.com/wealdtech/go-ens/v2"
)

var ensAddressSetAddressStr string

// ensAddressSetCmd represents the ens address set command
var ensAddressSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set the address of an ENS domain",
	Long: `Set the address of a name registered with the Ethereum Name Service (ENS).  For example:

    ethereal ens address set --domain=enstest.eth --address=0x1234...5678 --passphrase="my secret passphrase"

The keystore for the account that owns the name must be local (i.e. listed with 'get accounts list') and unlockable with the supplied passphrase.

This will return an exit status of 0 if the transaction is successfully submitted (and mined if --wait is supplied), 1 if the transaction is not successfully submitted, and 2 if the transaction is successfully submitted but not mined within the supplied time limit.`,
	Run: func(cmd *cobra.Command, args []string) {
		cli.Assert(!offline, quiet, "Offline mode not supported at current with this command")
		cli.Assert(ensDomain != "", quiet, "--domain is required")

		registry, err := ens.NewRegistry(client)
		cli.ErrCheck(err, quiet, "Cannot obtain ENS registry contract")

		// Fetch the owner of the name
		owner, err := registry.Owner(ensDomain)
		cli.ErrCheck(err, quiet, "Cannot obtain owner")
		cli.Assert(bytes.Compare(owner.Bytes(), ens.UnknownAddress.Bytes()) != 0, quiet, fmt.Sprintf("owner of %s is not set", ensDomain))
		outputIf(verbose, fmt.Sprintf("Domain is owned by %s", ens.Format(client, owner)))

		// Obtain the address
		address, err := ens.Resolve(client, ensAddressSetAddressStr)
		cli.Assert(bytes.Compare(address.Bytes(), ens.UnknownAddress.Bytes()) != 0, quiet, "Invalid address; if you are trying to clear an existing address use \"ens address clear\"")
		cli.ErrCheck(err, quiet, fmt.Sprintf("Invalid name/address %s", ensAddressSetAddressStr))

		// Obtain the resolver for this name
		resolver, err := ens.NewResolver(client, ensDomain)
		cli.ErrCheck(err, quiet, "No resolver for that name")
		outputIf(verbose, fmt.Sprintf("Resolver is %s", ens.Format(client, resolver.ContractAddr)))

		opts, err := generateTxOpts(owner)
		cli.ErrCheck(err, quiet, "Failed to generate transaction options")
		signedTx, err := resolver.SetAddress(opts, address)
		cli.ErrCheck(err, quiet, "Failed to send transaction")

		handleSubmittedTransaction(signedTx, log.Fields{
			"group":     "ens/address",
			"command":   "set",
			"ensdomain": ensDomain,
			"address":   address.Hex(),
		}, true)
	},
}

func init() {
	ensAddressCmd.AddCommand(ensAddressSetCmd)
	ensAddressFlags(ensAddressSetCmd)
	ensAddressSetCmd.Flags().StringVar(&ensAddressSetAddressStr, "address", "", "The name or address to which to resolve")
	addTransactionFlags(ensAddressSetCmd, "passphrase for the account that owns the domain")
}
