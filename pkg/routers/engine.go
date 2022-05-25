package routers

import (
	"github.com/gin-gonic/gin"
	"voucher/pkg/api"
)

func VoucherEngine() *gin.Engine {
	router := gin.New()

	userRouter := router.Group("/voucher/users")
	{
		userRouter.PATCH(":user", api.Redeem)
		userRouter.GET(":user/:seq", api.GetVoucher)
		userRouter.GET(":user/activities", api.ListVoucher)
	}

	adminRouter := router.Group("/voucher/admin")
	{
		adminRouter.POST("", api.CreateVoucher)
		adminRouter.GET("", api.CountVoucher)
	}

	return router
}