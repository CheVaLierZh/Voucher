package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"voucher/pkg/api/dbservice"
	"voucher/pkg/vouchercode"
)

// Redeem 兑换领奖码
func Redeem(ctx *gin.Context) {
	user := ctx.Param("user")
	code := ctx.PostForm("code")

	seq, c, valid := vouchercode.Decode(code)
	if !valid {
		ctx.JSON(http.StatusOK, gin.H{
			"status": false,
			"message": "invalid voucher code",
		})
	}

	vouchercode.Decode(code)
	err := dbservice.DoRedeem(user, seq, c)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"status": false,
			"message": err.Error(),
		})
	} else {
		ctx.JSON(http.StatusOK, gin.H{
			"status": true,
			"message": "redeem succeed",
		})
	}
}



// GetVoucher 获得领奖码
func GetVoucher(ctx *gin.Context) {
	seq, err := strconv.Atoi(ctx.Param("seq"))
	if err != nil {
		return
	}
	usr := ctx.Param("user")

	code, err := dbservice.DoGetVoucher(usr, seq)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"status": false,
			"message": err.Error(),
		})
	} else {
		ctx.JSON(http.StatusOK, gin.H{
			"status": true,
			"message": code,
		})
	}
}

// ListVoucher 列出用户的所有领奖码
func ListVoucher(ctx *gin.Context) {
	user := ctx.Param("user")

	info, _ := dbservice.DoListVoucher(user)

	ctx.JSON(http.StatusOK, gin.H{
		"status": "true",
		"list": info,
	})
}