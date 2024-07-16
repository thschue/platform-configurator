/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
)

var counters = map[string]int{}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Configures the deployment of the platform",
	Long:  `Configures the deployment of the platform`,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := cfg.Harbor.IsAvailable()
		if err != nil {
			counters["harbor"] = counters["harbor"] + 1
			fmt.Println("Harbor is not available, retrying")
			if counters["harbor"] > 10 {
				fmt.Println("Error connecting to harbor")
				log.Fatal("harbor is not available after 10 retries: %w", err)
			}
		}

		_, err = cfg.Gitea.IsAvailable()
		if err != nil {
			counters["gitea"] = counters["gitea"] + 1
			fmt.Println("Gitea is not available, retrying")
			if counters["gitea"] > 10 {
				fmt.Println("Error connecting to gitea")
				log.Fatal("gitea is not available after 10 retries: %w", err)
			}
		}

		for k, v := range cfg.Harbor.Configuration {
			data := map[string]interface{}{
				k: v,
			}
			err := cfg.Harbor.CreateConfiguration(data)
			if err != nil {
				log.Println("Error creating configuration: %w", err)
			}
		}

		for _, project := range cfg.Harbor.Projects {
			err := cfg.Harbor.CreateProject(project)
			if err != nil {
				log.Println("Error creating project: %w", err)
			}
		}

		for _, registry := range cfg.Harbor.Registries {
			err := cfg.Harbor.CreateRegistry(registry)
			if err != nil {
				log.Println("Error creating registry: %w", err)
			}
		}

		for _, rule := range cfg.Harbor.Replications {
			err := cfg.Harbor.CreateReplicationRule(rule)
			if err != nil {
				log.Println("Error creating replication rule: %w", err)
			}
		}

		for _, account := range cfg.Harbor.RobotAccounts {
			err := cfg.Harbor.CreateRobotAccount(account)
			if err != nil {
				log.Println("Error creating robot account: %w", err)
			}
		}

		for _, org := range cfg.Gitea.Orgs {
			err := cfg.Gitea.CreateOrganization(org)
			if err != nil {
				log.Println("Error creating organization: %w", err)
			}
		}

		for _, repo := range cfg.Gitea.Repositories {
			err := cfg.Gitea.CreateRepository(repo.Organization, repo)
			if err != nil {
				log.Println("Error creating repository: %w", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
