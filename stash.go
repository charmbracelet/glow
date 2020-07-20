package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	identityFile string
	forceKey     bool
	memo         string
	stashCmd     = &cobra.Command{
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
			fmt.Println("Stashed!")
			return nil
		},
	}

	stashListCmd = &cobra.Command{
		Use:    "stash-list",
		Hidden: false,
		Short:  "list your stashed markdowns",
		Long:   "",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			mds, err := cc.GetStash(1)
			if err != nil {
				return fmt.Errorf("error getting stash: " + err.Error())
			}

			const gap = " "
			const gapWidth = len(gap)
			numDigits := len(fmt.Sprintf("%d", len(mds)))
			termWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				return err
			}
			noteColWidth := termWidth - numDigits - gapWidth

			// Header
			fmt.Println("ID" + gap + "Note")
			fmt.Println(strings.Repeat("─", numDigits) + gap + strings.Repeat("─", noteColWidth))

			// Body
			for _, md := range mds {
				fmt.Printf("%"+fmt.Sprintf("%d", numDigits)+".d%s%s\n",
					md.ID,
					gap,
					runewidth.Truncate(md.Note, noteColWidth, "…"),
				)
			}

			return nil
		},
	}

	stashGetCmd = &cobra.Command{
		Use:    "stash-get",
		Hidden: false,
		Short:  "get a stashed markdown by id",
		Long:   "",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid markdown id")
			}
			cc := initCharmClient()
			md, err := cc.GetStashMarkdown(id)
			if err != nil {
				return fmt.Errorf("error getting markdown")
			}
			fmt.Println(md.Body)
			return nil
		},
	}

	stashDeleteCmd = &cobra.Command{
		Use:    "stash-delete",
		Hidden: false,
		Short:  "get a stashed markdown by id",
		Long:   "",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid markdown id")
			}
			cc := initCharmClient()
			err = cc.DeleteMarkdown(id)
			if err != nil {
				return fmt.Errorf("error deleting markdown")
			}
			fmt.Println("Deleted!")
			return nil
		},
	}
)

func getCharmConfig() *charm.Config {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	if identityFile != "" {
		cfg.SSHKeyPath = identityFile
		cfg.ForceKey = true
	}
	if forceKey {
		cfg.ForceKey = true
	}
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
