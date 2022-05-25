package dbservice

import (
	"context"
	"fmt"
	"github.com/tanimutomo/sqlfile"
	"log"
	"path/filepath"
	"strings"
	"time"
	"voucher/pkg/api/dbservice/mysql"
	"voucher/pkg/api/dbservice/redis"
	"voucher/pkg/config"
	"voucher/pkg/utils"
)

type Message struct {
	User string
	Code string
	Seq int
	Used bool
}

var MQChannel chan Message

func init() {
	mysql.InitMysqlClient(config.AppConf)
	redis.InitRedisClient(config.AppConf)
	err := InitMysqlDB(config.AppConf)
	if err != nil {
		log.Fatalln(err)
	}
	err = InitRedisDB(config.AppConf)
	if err != nil {
		log.Fatalln(err)
	}

	MQChannel = make(chan Message, config.AppConf.App.MessageQueueSize)
	go mQConsumer()
}

func mQConsumer() {
	for {
		msg := <- MQChannel

		if msg.Used {
			err := mysql.SetUsedVoucher(msg.Code)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			err := mysql.SetAllocatedVoucher(msg.User, msg.Code, msg.Seq)
			if err != nil {
				log.Println(err.Error())
			}
		}
	}
}

// InitMysqlDB 初始化Mysql，如果必要
func InitMysqlDB(conf *config.AppConfig) error {
	dotsqlDir := filepath.Join(utils.ExecFileDir(), "..")

	s := sqlfile.New()
	err := s.Files(filepath.Join(dotsqlDir, "activity.sql"), filepath.Join(dotsqlDir, "voucher.sql"),
		filepath.Join(dotsqlDir, "nextSeq.sql"))
	if err != nil {
		return err
	}
	_, err = s.Exec(mysql.Cli)
	return err
}

// InitRedisDB 缓存预热
// key type:
//          Hash: alivePeriod
//          Hash: redeemLimit
//          Set:   seq, set<code>    未领取
//          Set: usr-seq, set<code>    已领取记录
//          KV:   code, bool           使用情况
//          Hash: rest   <seq, rest>     期次剩余
//			Set: allSeqs set<seq>
func InitRedisDB(conf *config.AppConfig) error {
	conf = config.AppConf

	ctx := context.Background()
	ret := redis.Cli.DBSize(ctx)
	if ret.Val() != 0 {
		return nil
	}

	pipeline := redis.Cli.Pipeline()
	// 加载未过期的活动
	activities := mysql.GetActivities()
	aliveSeq := make(map[interface{}]bool)
	seqs := make(map[int]bool)
	now := time.Now()
	for _, x := range activities {
		if x.StartTime.Before(now) && x.EndTime.After(now) {
			aliveSeq[x.Seq] = true
			pipeline.HSet(ctx, "alivePeriod", fmt.Sprint(x.Seq), strings.Join([]string{x.StartTime.Format(conf.App.TimeFormat), x.EndTime.Format(conf.App.TimeFormat)}, ","))
			pipeline.HSet(ctx, "redeemLimit", fmt.Sprint(x.Seq), x.RedeemLimit)
			pipeline.HSet(ctx, "rest", fmt.Sprint(x.Seq), x.Rest)
		}
		seqs[x.Seq] = true
	}
	for k := range seqs {
		pipeline.SAdd(ctx, "allSeqs", k)
	}
	pipeline.Exec(ctx)

	// 加载未领取的领奖码
	tmp := make([]interface{}, 0, len(aliveSeq))
	for k := range aliveSeq {
		tmp = append(tmp, k)
	}
	vouchers := mysql.GetAliveUnallocatedVouchers(tmp, config.AppConf.App.Redis.NotAllocPreheatSize)
	for _, v := range vouchers {
		pipeline.SAdd(ctx, fmt.Sprintf("%d", v.Seq), v.Code)
	}
	pipeline.Exec(ctx)
	return nil
}