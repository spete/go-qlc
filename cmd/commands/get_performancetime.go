/*
 * Copyright (c) 2019 QLC Chain Team
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

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

	path1 := filepath.Join(cfgPath, "performanceTime.json")
	_ = os.Remove(path1)

	fd1, err := os.OpenFile(path1, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
	}

	path2 := filepath.Join(cfgPath, "performanceTime_orig.json")
	_ = os.Remove(path2)

	fd2, err := os.OpenFile(path2, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
	}

	var tmp Int64Slice

	loc, _ := time.LoadLocation("Asia/Shanghai")
	err = l.PerformanceTimes(func(p *types.PerformanceTime) {
		_, _ = fd2.WriteString(fmt.Sprintf("%s\n", p.String()))
		if p.T0 == 0 || p.T1 == 0 || p.T2 == 0 || p.T3 == 0 || p.Hash.IsZero() {
			//_, _ = fd1.WriteString(fmt.Sprintf("invalid data %s \n", p.String()))
		} else {
			t0 := time.Unix(0, p.T0)
			t1 := time.Unix(0, p.T1)
			d1 := t1.Sub(t0)
			t2 := time.Unix(0, p.T2)
			t3 := time.Unix(0, p.T3)
			d2 := t3.Sub(t2)
			tmp = append(tmp, d2.Nanoseconds())
			s := fmt.Sprintf("receive block[\"%s\"] @ %s, 1st consensus cost %s, full consensus cost %s\n", p.Hash.String(), time.Unix(0, p.T0).In(loc).Format("2006-01-02 15:04:05.000000"), d2, d1)
			_, _ = fd1.WriteString(s)
		}
	})

	sort.Sort(tmp)

	tmp2 := tmp[100 : len(tmp)-100]

	sum := int64(0)

	for _, v := range tmp2 {
		sum += v
	}

	t := fmt.Sprintf("%dns", sum/int64(len(tmp2)))
	duration, _ := time.ParseDuration(t)
	fmt.Println("avenge consensus", duration)

	if err != nil {
		fmt.Println(err)
	}
	return nil
}

type Int64Slice []int64

func (c Int64Slice) Len() int {
	return len(c)
}
func (c Int64Slice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Int64Slice) Less(i, j int) bool {
	return c[i] < c[j]
}
