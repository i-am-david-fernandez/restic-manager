// Copyright © 2018 David Fernandez <i.am.david.fernandez@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"

	"github.com/i-am-david-fernandez/glog"
	resticmanager "github.com/i-am-david-fernandez/restic-manager/internal"
	"github.com/spf13/cobra"
)

// sanityCmd represents the sanity command
var sanityCmd = &cobra.Command{
	Use:   "sanity",
	Short: "Perform a sanity check of the specified profile(s).",
	Long:  `Performs various checks to determine the saneness/validity of profile configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sanity called")

		for _, profile := range resticmanager.AppConfig.Profiles {
			glog.Infof("Processing profile %v", profile.Name())
			glog.Debugf("  from file %v", profile.File())

			glog.Infof("Performing sanity check.")

			errors := 0

			if !profile.SourceIsPresent() {
				glog.Errorf("  Source %v is not present.", profile.Source())
				errors++
			}

			if errors > 0 {
				glog.Errorf("Profile is problematic.")
			} else {
				glog.Infof("Profile is sane.")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(sanityCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sanityCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sanityCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
