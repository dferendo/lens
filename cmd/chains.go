package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/lens/client"
	"golang.org/x/sync/errgroup"
)

func chainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chains",
		Short: "manage local chain configurations",
	}

	cmd.AddCommand(
		cmdChainsAdd(),
		cmdChainsDelete(),
		cmdChainsEdit(),
		cmdChainsList(),
		cmdChainsShow(),
		cmdChainsSetDefault(),
		cmdChainsRegistryList(),
	)

	return cmd
}

func cmdChainsRegistryList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry-list",
		Args:    cobra.NoArgs,
		Aliases: []string{"rl"},
		Short:   "list chains available for configuration from the regitry",
		RunE: func(cmd *cobra.Command, args []string) error {
			tree, res, err := github.NewClient(nil).Git.GetTree(cmd.Context(), "cosmos", "chain-registry", "master", true)
			if err != nil || res.StatusCode != 200 {
				return err
			}
			chains := []string{}
			for _, entry := range tree.Entries {
				if *entry.Type == "tree" && !strings.Contains(*entry.Path, ".github") {
					chains = append(chains, *entry.Path)
				}
			}
			bz, err := json.Marshal(chains)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}
	return cmd
}

func cmdChainsAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add [[chain-name]]",
		Args:    cobra.MinimumNArgs(1),
		Aliases: []string{"a"},
		Short:   "add configraion for a chain or a number of chains from the chain registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			ch, err := getRegistryChain(args[0])
			if err != nil {
				return err
			}
			// al, err := getAssetFile(args[0])
			// if err != nil {
			// 	return err
			// }
			rpcs, err := ch.WorkingRPCs()
			if err != nil {
				return err
			}
			fmt.Println(rpcs)
			return nil
		},
	}
	return cmd
}

func cmdChainsDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{},
		Short:   "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func cmdChainsEdit() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "Edit",
		Aliases: []string{},
		Short:   "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func cmdChainsList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "List",
		Aliases: []string{},
		Short:   "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func cmdChainsShow() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "Show",
		Aliases: []string{},
		Short:   "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func cmdChainsSetDefault() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "SetDefault",
		Aliases: []string{},
		Short:   "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func getChainConfigFromRegistry(chainName, keyDirectory string, debug bool) (*client.ChainClientConfig, error) {
	rc, al, err := getChainFiles(chainName)
	if err != nil {
		return nil, err
	}
	var gasPrices string
	if len(al.Assets) > 0 {
		gasPrices = fmt.Sprintf("%d%s", 0.01, al.Assets[0].Base)
	}
	return &client.ChainClientConfig{
		Key:            "default",
		ChainID:        rc.ChainID,
		RPCAddr:        "",
		AccountPrefix:  rc.Bech32Prefix,
		KeyringBackend: "test",
		GasAdjustment:  1.2,
		GasPrices:      gasPrices,
		KeyDirectory:   keyDirectory,
		Debug:          debug,
		Timeout:        "20s",
		OutputFormat:   "json",
		BroadcastMode:  "block",
		SignModeStr:    "direct",
	}, nil
}

func getChainFiles(chainName string) (rc RegistryChain, al AssetList, err error) {
	eg := errgroup.Group{}
	eg.Go(func() (err error) {
		rc, err = getRegistryChain(chainName)
		return err
	})
	eg.Go(func() (err error) {
		al, err = getAssetFile(chainName)
		return err
	})
	err = eg.Wait()
	return
}

func getRegistryChain(chainName string) (RegistryChain, error) {
	var (
		cl       = github.NewClient(nil)
		ctx      = context.Background()
		registry RegistryChain
		file     = fmt.Sprintf("%s/chain.json", chainName)
		options  = &github.RepositoryContentGetOptions{}
	)

	ch, _, res, err := cl.Repositories.GetContents(ctx, "cosmos", "chain-registry", file, options)
	if err != nil || res.StatusCode != 200 {
		return registry, err
	}
	f, err := ch.GetContent()
	if err != nil {
		return registry, err
	}
	if err := json.Unmarshal([]byte(f), &registry); err != nil {
		return registry, err
	}
	return registry, nil
}

func getAssetFile(chainName string) (AssetList, error) {
	var (
		cl        = github.NewClient(nil)
		ctx       = context.Background()
		assetList AssetList
		file      = fmt.Sprintf("%s/assetlist.json", chainName)
		options   = &github.RepositoryContentGetOptions{}
	)

	ch, _, res, err := cl.Repositories.GetContents(ctx, "cosmos", "chain-registry", file, options)
	if err != nil || res.StatusCode != 200 {
		return assetList, err
	}
	f, err := ch.GetContent()
	if err != nil {
		return assetList, err
	}
	if err := json.Unmarshal([]byte(f), &assetList); err != nil {
		return assetList, err
	}
	return assetList, nil
}

func (rc RegistryChain) WorkingRPCs() ([]string, error) {
	var eg errgroup.Group
	rpcs, err := rc.GetRPCs()
	if err != nil {
		return nil, err
	}
	test := map[int]string{}
	for i, rpc := range rpcs {
		i, rpc := i, rpc
		eg.Go(func() error {
			cl, err := client.NewRPCClient(rpc, 5*time.Second)
			if err != nil {
				// return nil because we want to continue
				return nil
			}
			stat, err := cl.Status(context.Background())
			if err != nil {
				// return nil because we want to continue
				return nil
			}
			if !stat.SyncInfo.CatchingUp {
				test[i] = rpc
				return nil
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	var out []string
	for _, v := range test {
		out = append(out, v)
	}
	return out, nil
}

func (rc RegistryChain) GetRPCs() (out []string, err error) {
	for _, c := range rc.Apis.RPC {
		u, err := url.Parse(c.Address)
		if err != nil {
			return nil, err
		}
		var port string
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = u.Port()
		}
		out = append(out, fmt.Sprintf("%s://%s:%s", u.Scheme, u.Hostname(), port))
	}
	return
}

type RegistryChain struct {
	Schema       string `json:"$schema"`
	ChainName    string `json:"chain_name"`
	Status       string `json:"status"`
	NetworkType  string `json:"network_type"`
	PrettyName   string `json:"pretty_name"`
	ChainID      string `json:"chain_id"`
	Bech32Prefix string `json:"bech32_prefix"`
	DaemonName   string `json:"daemon_name"`
	NodeHome     string `json:"node_home"`
	Genesis      struct {
		GenesisURL string `json:"genesis_url"`
	} `json:"genesis"`
	Slip44   int `json:"slip44"`
	Codebase struct {
		GitRepo            string   `json:"git_repo"`
		RecommendedVersion string   `json:"recommended_version"`
		CompatibleVersions []string `json:"compatible_versions"`
	} `json:"codebase"`
	Peers struct {
		Seeds []struct {
			ID       string `json:"id"`
			Address  string `json:"address"`
			Provider string `json:"provider,omitempty"`
		} `json:"seeds"`
		PersistentPeers []struct {
			ID      string `json:"id"`
			Address string `json:"address"`
		} `json:"persistent_peers"`
	} `json:"peers"`
	Apis struct {
		RPC []struct {
			Address  string `json:"address"`
			Provider string `json:"provider"`
		} `json:"rpc"`
		Rest []struct {
			Address  string `json:"address"`
			Provider string `json:"provider"`
		} `json:"rest"`
	} `json:"apis"`
}

type AssetList struct {
	Schema  string `json:"$schema"`
	ChainID string `json:"chain_id"`
	Assets  []struct {
		Description string `json:"description"`
		DenomUnits  []struct {
			Denom    string `json:"denom"`
			Exponent int    `json:"exponent"`
		} `json:"denom_units"`
		Base     string `json:"base"`
		Name     string `json:"name"`
		Display  string `json:"display"`
		Symbol   string `json:"symbol"`
		LogoURIs struct {
			Png string `json:"png"`
			Svg string `json:"svg"`
		} `json:"logo_URIs"`
		CoingeckoID string `json:"coingecko_id"`
	} `json:"assets"`
}