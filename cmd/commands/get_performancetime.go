/*
 * Copyright (c) 2019 QLC Chain Team
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qlcchain/go-qlc/common/types"
	"github.com/qlcchain/go-qlc/config"

	"github.com/spf13/cobra"
)

// bcCmd represents the bc command
var performanceTimeCmd = &cobra.Command{
	Use:   "pt",
	Short: "get performance time",
	Run: func(cmd *cobra.Command, args []string) {
		err := getPerformanceTime()
		if err != nil {
			cmd.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(performanceTimeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bcCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bcCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getPerformanceTime() error {
	if cfgPath == "" {
		cfgPath = config.DefaultDataDir()
	}
	cm := config.NewCfgManager(cfgPath)
	cfg, err := cm.Load()
	if err != nil {
		return err
	}
	err = initNode(types.ZeroAddress, "", cfg)
	if err != nil {
		return err
	}
	l := ctx.Ledger.Ledger

	fd, err := os.OpenFile(filepath.Join(cfgPath, "performanceTime.json"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
	}
	hh := []byte("\r\n")
	var bytes []byte
	err = l.PerformanceTimes(func(p *types.PerformanceTime) {
		bytes, _ = json.Marshal(p)
		for i := 0; i < len(hh); i++ {
			bytes = append(bytes, hh[i])
		}
		_, _ = fd.Write(bytes)
	})
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
