package cmd

import (
	"fmt"
	"strings"

	vCore "github.com/InazumaV/V2bX/core"
	"github.com/spf13/cobra"
)

var (
	version  = "TempVersion" //use ldflags replace
	codename = "V2bX"
	intro    = "*"
)

var versionCommand = cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(_ *cobra.Command, _ []string) {
		showVersion()
	},
}

func init() {
	command.AddCommand(&versionCommand)
}

func showVersion() {
	fmt.Println(` 
  _/      _/    _/_/    _/        _/      _/   
 _/      _/  _/    _/  _/_/_/      _/  _/      
_/      _/      _/    _/    _/      _/         
 _/  _/      _/      _/    _/    _/  _/        
  _/      _/_/_/_/  _/_/_/    _/      _/        
                                                `)
	fmt.Printf("%s %s (%s) \n", codename, version, intro)
	fmt.Printf("二选一: %s\n", strings.Join(vCore.RegisteredCore(), ", "))
	// Warning
	fmt.Println(Warn("面板大于 >= 1.7.0."))
	fmt.Println(Warn("报错就改 /root/config.json"))
}
