package cmd

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/strangelove-ventures/lens/client/query"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FlagCommission = "commission"
	FlagAll        = "all"
)

// TODO: should this be [from] [validator-address]?
// if so then we should make the first arg mandatory and further args be []sdk.ValAddr
// and make the []sdk.ValAddr optional. This way we don't need any of the flags except
// commission
func distributionWithdrawRewardsCmd(lc *lensConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-rewards [validator-addr] [from]",
		Short: "Withdraw rewards from a given delegation address, and optionally withdraw validator commission if the delegation address given is a validator operator",
		Long: strings.TrimSpace(
			`Withdraw rewards from a given delegation address,
and optionally withdraw validator commission if the delegation address given is a validator operator.
Example:
$ lens tx withdraw-rewards cosmosvaloper1uyccnks6gn6g62fqmahf8eafkedq6xq400rjxr default
$ lens tx withdraw-rewards cosmosvaloper1uyccnks6gn6g62fqmahf8eafkedq6xq400rjxr default --commission
$ lens tx withdraw-rewards --from mykey --all
`,
		),
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()
			key := ""
			if len(args) == 1 {
				key = cl.Config.Key
			} else {
				key = args[1]
			}

			delAddr, err := cl.AccountFromKeyOrAddress(key)
			if err != nil {
				return err
			}
			encodedAddr := cl.MustEncodeAccAddr(delAddr)
			msgs := []sdk.Msg{}

			query := query.Query{Client: cl, Options: query.DefaultOptions()}
			if all, _ := cmd.Flags().GetBool(FlagAll); all {

				resp, err := query.DelegatorValidators(encodedAddr)
				if err != nil {
					return err
				}

				// build multi-message transaction
				for _, valAddr := range resp.Validators {
					val, err := cl.DecodeBech32ValAddr(valAddr)
					if err != nil {
						return err
					}
					msg := types.NewMsgWithdrawDelegatorReward(delAddr, sdk.ValAddress(val))
					msgs = append(msgs, msg)
				}

			} else if len(args) == 1 {
				valAddr, err := cl.DecodeBech32ValAddr(args[0])
				if err != nil {
					return err
				}
				msgs = append(msgs, types.NewMsgWithdrawDelegatorReward(delAddr, sdk.ValAddress(valAddr)))
			}

			if commission, _ := cmd.Flags().GetBool(FlagCommission); commission {
				valAddr, err := cl.DecodeBech32ValAddr(args[0])
				if err != nil {
					return err
				}
				msgs = append(msgs, types.NewMsgWithdrawValidatorCommission(sdk.ValAddress(valAddr)))
			}

			return cl.HandleAndPrintMsgSend(cl.SendMsgs(cmd.Context(), msgs))
		},
	}
	cmd.Flags().BoolP(FlagCommission, "c", false, "withdraw commission from a validator")
	cmd.Flags().BoolP(FlagAll, "a", false, "withdraw all rewards of a delegator")
	AddTxFlagsToCmd(cmd)
	return cmd
}

func distributionParamsCmd(lc *lensConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "query things about a chain's distribution params",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()

			params, err := cl.QueryDistributionParams(cmd.Context())
			if err != nil {
				return err
			}

			return cl.PrintObject(params)
		},
	}

	return cmd
}

func distributionCommunityPoolCmd(lc *lensConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "community-pool",
		Short: "query things about a chain's community pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()

			pool, err := cl.QueryDistributionCommunityPool(cmd.Context())
			if err != nil {
				return err
			}

			return cl.PrintObject(pool)
		},
	}

	return cmd
}

func distributionCommissionCmd(lc *lensConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commission [validator-address]",
		Args:  cobra.ExactArgs(1),
		Short: "query a specific validator's commission",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()
			address, err := cl.DecodeBech32ValAddr(args[0])
			if err != nil {
				return err
			}

			commission, err := cl.QueryDistributionCommission(cmd.Context(), address)
			if err != nil {
				return err
			}

			return cl.PrintObject(commission)
		},
	}

	return cmd
}

func distributionRewardsCmd(lc *lensConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rewards [key-or-delegator-address] [validator-address]",
		Short: "query things about a delegator's rewards",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()
			delAddr, err := cl.AccountFromKeyOrAddress(args[0])
			if err != nil {
				return err
			}

			valAddr, err := cl.DecodeBech32ValAddr(args[1])
			if err != nil {
				return err
			}

			rewards, err := cl.QueryDistributionRewards(cmd.Context(), delAddr, valAddr)
			if err != nil {
				return err
			}

			return cl.PrintObject(rewards)
		},
	}

	return cmd
}

func distributionSlashesCmd(v *viper.Viper, lc *lensConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slashes [validator-address] [start-height] [end-height]",
		Short: "query things about a validator's slashes on a chain",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()

			pageReq, err := ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			address, err := cl.DecodeBech32ValAddr(args[0])
			if err != nil {
				return err
			}

			startHeight, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			endHeight, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			slashes, err := cl.QueryDistributionSlashes(cmd.Context(), address, startHeight, endHeight, pageReq)
			if err != nil {
				return err
			}

			return cl.PrintObject(slashes)
		},
	}

	return paginationFlags(cmd, v)
}

func distributionValidatorRewardsCmd(lc *lensConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-outstanding-rewards [address]",
		Short: "query things about a validator's (and all their delegators) outstanding rewards on a chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()

			address, err := cl.DecodeBech32ValAddr(args[0])
			if err != nil {
				return err
			}

			rewards, err := cl.QueryDistributionValidatorRewards(cmd.Context(), address)
			if err != nil {
				return err
			}

			return cl.PrintObject(rewards)
		},
	}
	return cmd
}

func distributionDelegatorValidatorsCmd(lc *lensConfig) *cobra.Command {
	var delegator string
	cmd := &cobra.Command{
		Use:     "delegator-validators [delegator_address]",
		Aliases: []string{"dv", "delval"},
		Short:   "query the delegator's validators",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				delegator = ""
				return nil
			}
			if len(args) != 1 {
				cmd.Usage()
				return fmt.Errorf("\n please specify the delegator's address")
			} else {
				delegator = args[0]
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := lc.config.GetDefaultClient()
			// Check if the address has a valid format
			if len(delegator) > 0 {
				_, err := cl.DecodeBech32AccAddr(delegator)
				if err != nil {
					return fmt.Errorf("\n please specify a valid delegator's address for chain '%s'. Address should start with '%s'", cl.Config.ChainID, cl.Config.AccountPrefix)
					return err
				}
			}

			address, err := cl.AccountFromKeyOrAddress(delegator)
			if err != nil {
				return err
			}
			encodedAddr := cl.MustEncodeAccAddr(address)

			// Query options
			pr, err := ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			height, err := ReadHeight(cmd.Flags())
			if err != nil {
				return err
			}

			options := query.QueryOptions{Pagination: pr, Height: height}
			query := query.Query{Client: cl, Options: &options}
			delValidators, err := query.DelegatorValidators(encodedAddr)
			if err != nil {
				return err
			}
			return cl.PrintObject(delValidators)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
