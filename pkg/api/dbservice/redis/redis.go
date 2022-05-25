package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"strings"
	"time"
	"voucher/pkg/api/dbservice/mysql"
	"voucher/pkg/config"
)

var Cli *redis.Client

func InitRedisClient(conf *config.AppConfig) {
	Cli = redis.NewClient(&redis.Options{
		Addr: conf.App.Redis.Address,
		Password: conf.App.Redis.Password,
		DB: conf.App.Redis.Dbname, // use default db
	})
}

func CacheAllocatedVouchers(usr string, vouchers []mysql.Voucher) {
	expireKeys := make(map[string]bool)
	ctx := context.Background()
	for _, v := range vouchers {
		key := fmt.Sprintf("%s-%d", usr, v.Seq)
		Cli.SAdd(ctx, key, v.Code)
		expireKeys[key] = true
		Cli.Set(ctx, v.Code, v.Used, time.Duration(config.AppConf.App.Redis.ExpireTime) * time.Second)
	}
	for k := range expireKeys {
		Cli.Expire(ctx, k, time.Duration(config.AppConf.App.Redis.ExpireTime) * time.Second)
	}
}

func CacheActivities(activities []mysql.Activity) {
	ctx := context.Background()
	conf := config.AppConf

	for _, x := range activities {
		Cli.HSet(ctx, "alivePeriod", fmt.Sprint(x.Seq), strings.Join([]string{x.StartTime.Format(conf.App.TimeFormat), x.EndTime.Format(conf.App.TimeFormat)}, ","))
		Cli.HSet(ctx, "redeemLimit", fmt.Sprint(x.Seq), x.RedeemLimit)
		Cli.HSet(ctx, "rest", fmt.Sprint(x.Seq), x.Rest)
	}
}

const redeemScript = `
	-- redeem code
	-- KEYS[1]: code
    -- return 0, failed
    -- return 1, success
    local used = redis.call("GET", KEYS[1])
	if used == "false" 
	then
		redis.call("SET", KEYS[1], "true")
		return 1
	end
	return 0
`

func DoRedeem(code string) bool {
	script := redis.NewScript(redeemScript)
	ret, _ := script.Run(context.Background(), Cli, []string{code}).Int()
	if ret == 1 {
		return true
	} else {
		return false
	}
}

const getVoucherScript = `
	-- KEYS[1]: usr-seq
	-- KEYS[2]: seq
	-- return "-1", has rest but no code
	-- return "-2", number of voucher reach limit
	-- return "-3", no rest
    -- return "xxx", success
	local rest = redis.call("HGET", "rest", KEYS[2])
	if tonumber(rest) == 0 
	then 
		return "-3"
	end
	local n = redis.call("SCARD", KEYS[1])
	local limit = redis.call("HGET", "rest", KEYS[2])
	if n < tonumber(limit) 
	then 
		local code = redis.call("SPOP", KEYS[2])
		if code == false
		then 
			return "-1"
		end
		return code
	end
	return "-2"
`

func DoGetVoucher(usr string, seq int) (string, error) {
	script := redis.NewScript(getVoucherScript)
	ret := script.Run(context.Background(), Cli, []string{fmt.Sprintf("%s-%d", usr, seq), fmt.Sprint(seq)}).String()
	if ret == "-2" {
		return ret, errors.New("number of vouchers of user in this seq reach limit")
	} else if ret == "-1" {
		return ret, nil
	} else if ret == "-3" {
		return ret, errors.New("no rest vouchers")
	} else {
		return ret, nil
	}
}

func CacheNotAllocatedVouchers(vouchers []mysql.Voucher) {
	ctx := context.Background()
	for _, v := range vouchers {
		Cli.SAdd(ctx, fmt.Sprint(v.Seq), v.Code)
	}
}