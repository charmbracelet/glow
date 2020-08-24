package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var (
	/*
		identityFile string
		forceKey     bool
	*/

	memo     string
	stashCmd = &cobra.Command{
		Use:    "stash SOURCE",
		Hidden: false,
		Short:  "stash a markdown",
		Long:   "",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("bad filename")
			}
			defer f.Close()
			b, err := ioutil.ReadAll(f)
			if err != nil {
				return fmt.Errorf("error reading file")
			}
			err = cc.StashMarkdown(memo, string(b))
			if err != nil {
				return fmt.Errorf("error stashing markdown")
			}
			dot := termenv.String("•").Foreground(common.Green.Color()).String()
			fmt.Println(dot + " Stashed!")
			return nil
		},
	}
)

func getCharmConfig() *charm.Config {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	/*
		if identityFile != "" {
			cfg.SSHKeyPath = identityFile
			cfg.ForceKey = true
		}
		if forceKey {
			cfg.ForceKey = true
		}
	*/

	return cfg
}

func initCharmClient() *charm.Client {
	cfg := getCharmConfig()
	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		log.Fatal("Missing ssh key. Run `ssh-keygen` to make one or set the `CHARM_SSH_KEY_PATH` env var to your private key path.")
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cc
}
