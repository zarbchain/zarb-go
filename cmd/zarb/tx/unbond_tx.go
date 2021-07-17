package tx

import (
	"fmt"

	cli "github.com/jawher/mow.cli"
	"github.com/zarbchain/zarb-go/cmd"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
	grpcclient "github.com/zarbchain/zarb-go/www/grpc/client"
)

func UnbondTx() func(c *cli.Cmd) {
	return func(c *cli.Cmd) {
		stampOpt := c.String(cli.StringOpt{
			Name: "stamp",
			Desc: "Transaction stamp",
		})

		seqOpt := c.Int(cli.IntOpt{
			Name: "seq",
			Desc: "Transaction sequence number",
		})

		valOpt := c.String(cli.StringOpt{
			Name: "val",
			Desc: "Validator's address",
		})

		memoOpt := c.String(cli.StringOpt{
			Name:  "memo",
			Desc:  "Transaction memo",
			Value: "",
		})

		authOpt := c.String(cli.StringOpt{
			Name: "a auth",
			Desc: "Passphrase of the key file",
		})
		keyFileOpt := c.String(cli.StringOpt{
			Name: "k keyfile",
			Desc: "Path to the encrypted key file",
		})

		grpcOpt := c.String(cli.StringOpt{
			Name: "e endpoint",
			Desc: "gRPC server address",
		})
		c.Before = func() { fmt.Println(cmd.ZARB) }
		c.Action = func() {

			var err error
			var stamp crypto.Hash
			var validator crypto.Address
			var seq int
			var auth string

			// ---
			if *valOpt == "" {
				cmd.PrintWarnMsg("Validator address is not defined.")
				c.PrintHelp()
				return
			}
			validator, err = crypto.AddressFromString(*valOpt)
			if err != nil {
				cmd.PrintErrorMsg("Validator address is not valid: %v", err)
				return
			}

			//sign transaction
			if *keyFileOpt == "" {
				cmd.PrintWarnMsg("Please specify a key file to sign.")
				c.PrintHelp()
				return
			}
			if *authOpt == "" {
				auth = cmd.PromptPassphrase("Passphrase: ", false)
			} else {
				auth = *authOpt
			}

			//RPC
			if seqOpt != nil {
				seq = *seqOpt
			} else {
				seq, err = grpcclient.GetSequence(promptRPCEndpoint(grpcOpt), validator)
				if err != nil {
					cmd.PrintErrorMsg("Couldn't retrieve sequence number from RPC Server: %v", err)
					return
				}
			}
			if stampOpt == nil || *stampOpt == "" {
				stamp, err = grpcclient.GetStamp(promptRPCEndpoint(grpcOpt))
				if err != nil {
					cmd.PrintErrorMsg("Couldn't retrieve stamp from RPC Server: %v", err)
					return
				}
			} else {
				stamp, err = crypto.HashFromString(*stampOpt)
				if err != nil {
					cmd.PrintErrorMsg("Couldn't decode stamp from input: %v", err)
					return
				}
			}

			trx := tx.NewUnbondTx(stamp, seq, validator, *memoOpt)

			signAndPublish(trx, *keyFileOpt, auth, grpcOpt)

		}
	}
}
