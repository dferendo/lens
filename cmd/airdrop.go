package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/lens/client"
)

func airdropCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "airdrop [airdrop.json] [denom] [key]?",
		Short: "Airdrop coins to a specified address",
		Long:  "The airdrop file consists of map[string]float64 where the key is the address on the target chain and the value is the amount of coins to be airdropped to that address/1e6 (i.e. atom instead of uatom). The airdrop command 1. checks the addresses in the file to ensure that they are valid for the given chain l",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			osmosis, _ := client.NewChainClient(client.GetOsmosisConfig("foo", false), "foo", os.Stdin, os.Stdout)
			cl := config.GetDefaultClient()
			keyNameOrAddress := ""
			if len(args) == 2 {
				keyNameOrAddress = cl.Config.Key
			} else {
				keyNameOrAddress = args[2]
			}
			address, err := cl.AccountFromKeyOrAddress(keyNameOrAddress)
			if err != nil {
				return err
			}

			f, err := os.Open(args[0])
			if err != nil {
				return err
			}
			bz, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			var airdrop airdropFile
			if err := json.Unmarshal(bz, &airdrop); err != nil {
				return err
			}

			multiMsg := &banktypes.MsgMultiSend{
				Inputs:  []banktypes.Input{},
				Outputs: []banktypes.Output{},
			}
			amount := sdk.Coin{Denom: args[1], Amount: sdk.NewInt(0)}
			var sent int
			for k, v := range airdrop {
				to, err := osmosis.DecodeBech32AccAddr(k)
				if err != nil {
					return err
				}
				toSendCoin := sdk.NewCoin(args[1], sdk.NewInt(int64(v*1e6)))
				toSend := sdk.NewCoins(toSendCoin)
				amount = amount.Add(toSendCoin)
				multiMsg.Outputs = append(multiMsg.Outputs, banktypes.NewOutput(to, toSend))
				sent += 1

				if len(multiMsg.Outputs) > 300 {
					completion := float64(sent) / float64(len(airdrop))
					fmt.Printf("(%f) sending %s to %d addresses\n", completion, amount.String(), len(multiMsg.Outputs))
					multiMsg.Inputs = append(multiMsg.Inputs, banktypes.NewInput(address, sdk.NewCoins(amount)))
					res, err := cl.SendMsgs(cmd.Context(), []sdk.Msg{multiMsg})
					if err != nil || res.Code != 0 {
						return err
					}
					multiMsg.Inputs = []banktypes.Input{}
					multiMsg.Outputs = []banktypes.Output{}
					amount = sdk.Coin{Denom: args[1], Amount: sdk.NewInt(0)}
				}
			}
			return nil
		},
	}
	cmd.Flags().Int("max-msgs", 20, "max number of msgs per tx to send")
	return cmd
}

type airdropFile map[string]float64
