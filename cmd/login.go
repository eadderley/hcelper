/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type VaultLDAPResponse struct {
	RequestID     string `json:"request_id"`
	LeaseID       string `json:"lease_id"`
	Renewable     bool   `json:"renewable"`
	LeaseDuration int    `json:"lease_duration"`
	Data          struct {
	} `json:"data"`
	WrapInfo interface{} `json:"wrap_info"`
	Warnings interface{} `json:"warnings"`
	Auth     struct {
		ClientToken   string   `json:"client_token"`
		Accessor      string   `json:"accessor"`
		Policies      []string `json:"policies"`
		TokenPolicies []string `json:"token_policies"`
		Metadata      struct {
			Username string `json:"username"`
		} `json:"metadata"`
		LeaseDuration int    `json:"lease_duration"`
		Renewable     bool   `json:"renewable"`
		EntityID      string `json:"entity_id"`
		TokenType     string `json:"token_type"`
		Orphan        bool   `json:"orphan"`
	} `json:"auth"`
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with your organization's auth provider",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PreRun: func(cmd *cobra.Command, args []string) {

	},
	Run: func(cmd *cobra.Command, args []string) {

		// Grab flag info
		username := cmd.Flag("username").Value.String()
		method := cmd.Flag("method").Value.String()
		env := cmd.Flag("env").Value.String()

		/* previously used for promptui.Prompt's validate
		envCheck := func(input string) error {
			_, err := regexp.MatchString(`pre|prod`, input)
			if err != nil {
				return errors.New(`env must be "pre" or "prod"`)
			}
			return nil
		}
		*/

		// Force env selection if not set
		if env == "" {
			envPrompt := promptui.Select{
				Label: "Select you environment",
				Items: []string{"pre", "prod"},
			}
			_, selectEnv, err := envPrompt.Run()
			if err != nil {
				fmt.Printf("Env input failed %v\n", err)
			}
			env = selectEnv
		}

		envUrl := viper.GetViper().GetString(env)

		os.Setenv("VAULT_ADDR", envUrl)

		// create the password prompt
		validate := func(input string) error {
			if len(input) < 6 {
				return errors.New("password must have more than 6 characters")
			}
			return nil
		}

		passPrompt := promptui.Prompt{
			Label:    "Password",
			Validate: validate,
			Mask:     '*',
		}

		password, err := passPrompt.Run()

		if err != nil {
			fmt.Printf("Password input failed %v\n", err)
			os.Exit(1)
		}
		submitPass := []byte(`{"password" : "` + password + `"}`)

		if err != nil {
			fmt.Printf("Error in parsing auth request parameters: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s/v1/auth/ldap/login/%s", envUrl, username)

		// Login to vault
		if method != "" {
			switch method {
			case "ldap":
				req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/auth/ldap/login/%s", envUrl, username), bytes.NewBuffer(submitPass))

				if err != nil {
					fmt.Printf("Error constructing LDAP login request: %v\n", err)
					os.Exit(1)
				}
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					fmt.Printf("Error logging into Vault via LDAP: %v\n", err)
					os.Exit(1)
				}
				defer resp.Body.Close()

				ldapResp := VaultLDAPResponse{}
				//var ldapResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&ldapResp)

				os.Setenv("VAULT_ADDR", envUrl)
				os.Setenv("VAULT_TOKEN", ldapResp.Auth.ClientToken)

				fmt.Printf("export VAULT_ADDR=%s\n", envUrl)
				fmt.Printf("export VAULT_TOKEN=%s\n", ldapResp.Auth.ClientToken)

			}
		}

	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().StringP("gateway", "g", "", "The Mortar API gateway URL")
	loginCmd.Flags().StringP("username", "u", "", "The username for user login credentials")
	loginCmd.Flags().StringP("env", "e", "", "The environment you're logging into (pre or prod)")
	loginCmd.Flags().StringP("method", "m", "ldap", "The login method")
	loginCmd.MarkFlagRequired("username")
}
