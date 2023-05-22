package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// configFile is a string path pointing to a config
	// file on disk.
	configFile string
	// certificate file.
	//
	// Used only when manual certificate/key configurations
	// are supplied.
	manCertFile = "cert.pem"
	// manKeyFile is a string path pointing to the x509
	// certificate key file.
	//
	// Used only when manual certificate/key configurations
	// are supplied.
	manKeyFile = "key.pem"
	// genAliases is a slice of aliases that point to
	// various generate-config subcommands.
	genAliases = []string{"gen", "gen-c"}
	// RootCmd is the base command for the CLI.
	RootCmd = &cobra.Command{
		Use:   "skyhook",
		Short: "Obfuscated HTTPS file transfer server.",
		Long:  "Obfuscated HTTPS file transfer server.",
		RunE:  nil,
	}
)
