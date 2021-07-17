package tx

import (
	"fmt"

	cli "github.com/jawher/mow.cli"
	"github.com/zarbchain/zarb-go/cmd"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
	grpcclient "github.com/zarbchain/zarb-go/www/grpc/client"
)

func WithdrawTx() func(c *cli.Cmd) {
	return func(c *cli.Cmd) {
		stampOpt := c.String(cli.StringOpt{
			Name: "stamp",
			Desc: "Transaction stamp",
		})

		seqOpt := c.Int(cli.IntOpt{
			Name: "seq",
			Desc: "Transaction sequence number",
		})

		fromOpt := c.String(cli.StringOpt{
			Name: "from",
			Desc: "withdraw from Validator address",
		})

		toOpt := c.String(cli.StringOpt{
			Name: "to",
			Desc: "Deposit to address",
		})

		amountOpt := c.Int(cli.IntOpt{
			Name: "amount",
			Desc: "The amount to be transferred",
		})

		feeOpt := c.Int(cli.IntOpt{
			Name: "fee",
			Desc: "Transaction fee",
		})

		memoOpt := c.String(cli.StringOpt{
			Name:  "memo",
			Desc:  "Transaction memo (Optional)",
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
			var from crypto.Address
			var to crypto.Address
			var seq int
			var amount int64
			var fee int64
			var auth string

			// ---
			if *amountOpt == 0 {
				cmd.PrintWarnMsg("Amount is not defined.")
				c.PrintHelp()
				return
			}
			amount = int64(*amountOpt)

			if *feeOpt == 0 {
				cmd.PrintWarnMsg("Fee is not defined.")
				c.PrintHelp()
				return
			}
			fee = int64(*feeOpt)

			if *fromOpt == "" {
				cmd.PrintWarnMsg("Validator address is not defined.")
				c.PrintHelp()
				return
			}
			from, err = crypto.AddressFromString(*fromOpt)
			if err != nil {
				cmd.PrintErrorMsg("Validator address is not valid: %v", err)
				return
			}

			if *toOpt == "" {
				cmd.PrintWarnMsg("Deposit to address is not defined.")
				c.PrintHelp()
				return
			}
			to, err = crypto.AddressFromString(*toOpt)
			if err != nil {
				cmd.PrintErrorMsg("Deposit to address is not valid: %v", err)
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
				seq, err = grpcclient.GetSequence(promptRPCEndpoint(grpcOpt), from)
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

			//fulfill transaction payload
			trx := tx.NewWithdrawTx(stamp, seq, from, to, amount, fee, *memoOpt)

			signAndPublish(trx, *keyFileOpt, auth, grpcOpt)
		}
	}
}
