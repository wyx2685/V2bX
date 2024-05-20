package main

import (
	//"net/http"
	//_ "net/http/pprof"

	"github.com/InazumaV/V2bX/cmd"
)

func main() {
	//内存泄漏排查
	//go func() {
	//	http.ListenAndServe("127.0.0.1:6060", nil)
	//}()
	cmd.Run()
}
