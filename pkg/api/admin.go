package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
	"voucher/pkg/api/dbservice"
	"voucher/pkg/config"
	"voucher/pkg/vouchercode"
)

func CreateVoucher(ctx *gin.Context) {
	n, err := strconv.Atoi(ctx.PostForm("number"))
	if err != nil {
		return
	}
	limit, err := strconv.Atoi(ctx.PostForm("redeemLimit"))
	if err != nil {
		return
	}
	startTime, err := time.Parse(config.AppConf.App.TimeFormat, ctx.PostForm("startTime"))
	if err != nil {
		return
	}
	endTime, err := time.Parse(config.AppConf.App.TimeFormat, ctx.PostForm("endTime"))
	if err != nil {
		return
	}
	description := ctx.PostForm("description")

	seq := dbservice.GetNextSeq()

	codes := vouchercode.New().Generate(seq, n)
	err = dbservice.InsertNewVouchers(seq, codes)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H {
			"status": false,
			"message": err.Error(),
		})
		return
	}

	err = dbservice.InsertNewActivity(seq, startTime, endTime, limit, n, description)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H {
			"status": false,
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H {
		"status": true,
		"message": "succeed",
	})
}


// CountVoucher 统计某一期领奖码的使用情况：领取数，兑换数，总数
func CountVoucher(ctx *gin.Context) {
	seq, err := strconv.Atoi(ctx.Param("seq"))
	if err != nil {
		return
	}

	total, allocated := dbservice.GetSeqTotalAndRest(seq)
	used := dbservice.GetSeqUsed(seq)

	ctx.JSON(http.StatusOK, gin.H{
		"status": true,
		"total": total,
		"allocated": allocated,
		"used": used,
	})
}
