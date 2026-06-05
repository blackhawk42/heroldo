package main

import (
	"fmt"

	"github.com/blackhawk42/heroldo/pkg/heroldo/registries"
	"github.com/spf13/cobra"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage authentication tokens",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := openRegistry()
		if err != nil {
			return err
		}
		defer registry.Close()

		entries, err := registry.ListTokens()
		if err != nil {
			return err
		}
		for _, entry := range entries {
			fmt.Printf("%s  %x\n", entry.Username, []byte(entry.Token))
		}
		return nil
	},
}

var createCmd = &cobra.Command{
	Use:   "create <username>",
	Short: "Create a new token for a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := openRegistry()
		if err != nil {
			return err
		}
		defer registry.Close()

		username := args[0]
		token, err := registry.NewToken(username)
		if err != nil {
			return err
		}
		fmt.Println(token)
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a token by username or by token value",
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := openRegistry()
		if err != nil {
			return err
		}
		defer registry.Close()

		username, _ := cmd.Flags().GetString("username")
		token, _ := cmd.Flags().GetString("token")
		if username == "" && token == "" {
			return fmt.Errorf("either --username or --token is required")
		}
		if username != "" && token != "" {
			return fmt.Errorf("--username and --token are mutually exclusive")
		}

		if username != "" {
			if err := registry.DeleteTokenByUsername(username); err != nil {
				return err
			}
			fmt.Printf("Token for user %s deleted\n", username)
		} else {
			if err := registry.DeleteTokenByToken(token); err != nil {
				return err
			}
			fmt.Printf("Token %s deleted\n", token)
		}
		return nil
	},
}

func init() {
	deleteCmd.Flags().String("username", "", "Delete by username")
	deleteCmd.Flags().String("token", "", "Delete by token value")
	tokensCmd.AddCommand(listCmd, createCmd, deleteCmd)
}

// openRegistry opens the token registry from the --auth-db flag.
func openRegistry() (*registries.BBoltTokenRegistry, error) {
	authDBPath := v.GetString("auth-db")
	if authDBPath == "" {
		return nil, fmt.Errorf("auth-db must be set to manage tokens")
	}
	return registries.NewBBoltTokenRegistry(authDBPath, nil, 0)
}
