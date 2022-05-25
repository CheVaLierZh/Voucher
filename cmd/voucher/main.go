package main

import (
	"fmt"
	"voucher/pkg/config"
	"voucher/pkg/routers"
)

func main() {
	router := routers.VoucherEngine()
	if err := router.Run(fmt.Sprintf(":%d", config.AppConf.App.Port)); err != nil {
		println("Error when running server. " + err.Error())
	}
}
