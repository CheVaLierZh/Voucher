package dbservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"voucher/pkg/api/dbservice/mysql"
	"voucher/pkg/api/dbservice/redis"
	"voucher/pkg/config"
)

func GetSeqAlivePeriod(seq int) (time.Time, time.Time, error) {
	ctx := context.Background()
	exist := redis.Cli.HExists(ctx, "alivePeriod", fmt.Sprint(seq)).Val()
	if !exist {
		activities := mysql.GetActivities()
		redis.CacheActivities(activities)
	}
	period := redis.Cli.HGet(ctx, "alivePeriod", fmt.Sprint(seq)).Val()
	splits := strings.Split(period, ",")
	if len(splits) != 2 {
		return time.Time{}, time.Time{}, errors.New("seq number has expired")
	}
	startTime, _ := time.Parse(config.AppConf.App.TimeFormat, splits[0])
	endTime, _ := time.Parse(config.AppConf.App.TimeFormat, splits[1])
	return startTime, endTime, nil
}

func CheckCodeIsAllocated(seq int, code string, usr string) bool {
	ctx := context.Background()

	key := fmt.Sprintf("%s-%d", usr, seq)
	existInt := redis.Cli.Exists(ctx, key).Val()
	if existInt == 0 {
		vouchers := mysql.GetUserSeqVoucher(usr, seq)
		redis.CacheAllocatedVouchers(usr, vouchers)
	}

	ok := redis.Cli.SIsMember(ctx, key, code).Val()
	return ok
}

func GetNotAllocatedVouchers(seq int) {
	vouchers := mysql.GetAliveUnallocatedVouchers([]interface{}{seq}, -1)
	redis.CacheNotAllocatedVouchers(vouchers)
}

func DoRedeem(usr string, seq int, code string) error {
	startTime, endTime, err := GetSeqAlivePeriod(seq)
	if err != nil {
		return err
	}

	now := time.Now()
	if !(startTime.Before(now) && endTime.After(now)) {
		return errors.New("voucher has expired")
	}

	ok := CheckCodeIsAllocated(seq, code, usr)
	if !ok {
		return errors.New("this code not allocated to this user")
	}

	ret := redis.DoRedeem(code)
	if ret {
		go func() {
			MQChannel <- Message{Code: code}
		}()
		return nil
	} else {
		return errors.New("this code has redeemed")
	}
}

func DoGetVoucher(usr string, seq int) (string, error) {
	ret, err := redis.DoGetVoucher(usr, seq)
	if err == nil && ret == "-1" {
		GetNotAllocatedVouchers(seq)
		return redis.DoGetVoucher(usr, seq)
	}
	if err == nil {
		go func() {
			MQChannel <- Message{Code: ret, User: usr, Seq: seq, Used: false}
		} ()
	}

	return ret, err
}

type voucherInfo struct {
	Code string	`json:"code"`
	StartTime string 	`json:"startTime"`
	EndTime string	`json:"endTime"`
}

func DoListVoucher(usr string) ([]byte, error) {
	ctx := context.Background()
	seqs := redis.Cli.SMembers(ctx, "allSeqs").Val()
	seqNotExist := make([]int, 0)
	keys := make([]string, 0, len(seqs))
	for _, seq := range seqs {
		key := fmt.Sprintf("%s-%d", usr, seq)
		keys = append(keys, key)
		ok := redis.Cli.Exists(ctx, key).Val()
		if ok == 0 {
			tmp, _ := strconv.Atoi(seq)
			seqNotExist = append(seqNotExist, tmp)
		}
	}

	vouchers := mysql.GetUserSeqVoucher(usr, seqNotExist...)
	redis.CacheAllocatedVouchers(usr, vouchers)

	res := make([]voucherInfo, 0)
	for _, k := range keys {
		seq, _ := strconv.Atoi(strings.Split(k, "-")[1])
		codes := redis.Cli.SMembers(ctx, k).Val()
		period := redis.Cli.HGet(ctx, "alivePeriod", fmt.Sprint(seq)).Val()
		splits := strings.Split(period, ",")
		for _, c := range codes {
			res = append(res, voucherInfo{Code: c, StartTime: splits[0], EndTime: splits[1]})
		}
	}

	return json.MarshalIndent(res, "", "\t")
}

func InsertNewVouchers(seq int, codes []string) error {
	str := make([]string, 0, len(codes))
	args := make([]interface{}, 0, len(codes))
	for _, c := range codes {
		str = append(str, "(?, ?, ?)")
		args = append(args, seq)
		args = append(args, c)
		args = append(args, "")
	}
	stmt := fmt.Sprintf("INSERT INTO voucher (seq, code, usr) VALUES %s", strings.Join(str, ","))
	_, err := mysql.Cli.Exec(stmt, args...)

	for _, code := range codes {
		redis.Cli.SAdd(context.Background(), fmt.Sprint(seq), code)
	}
	return err
}

func InsertNewActivity(seq int, startTime, endTime time.Time, limit, total int, description string) error {
	stmt := fmt.Sprintf("INSERT INTO activity (seq, startTime, endTime, redeemLimit, total, description) VALUES " +
		"(%d, %s, %s, %d, %d, %s)",
		seq, startTime.String(), endTime.String(), limit, total, description)
	_, err := mysql.Cli.Exec(stmt)

	ctx := context.Background()
	redis.Cli.SAdd(ctx, "allSeqs", seq)
	redis.Cli.HSet(ctx, "alivePeriod", seq, startTime.Format(config.AppConf.App.TimeFormat) + "," + endTime.Format(config.AppConf.App.TimeFormat))
	redis.Cli.HSet(ctx, "redeemLimit", seq, limit)
	redis.Cli.HSet(ctx, "rest", seq, total)
	return err
}

func GetNextSeq() int {
	tx, _ := mysql.Cli.Begin()
	_, err := tx.Exec("UPDATE nextSeq SET seq = seq + 1 WHERE id = 0")
	if err != nil {
		tx.Rollback()
		return GetNextSeq()
	}
	row := tx.QueryRow("SELECT seq FROM nextSeq WHERE id = 0")
	var seq int
	row.Scan(&seq)

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return GetNextSeq()
	}
	return seq
}

func GetSeqTotalAndRest(seq int) (int, int) {
	var total int
	var allocated int
	row := mysql.Cli.QueryRow(fmt.Sprintf("SELECT total, rest FROM activity WHERE seq = %d", seq))
	row.Scan(&total, &allocated)
	allocated = total - allocated
	return total, allocated
}

func GetSeqUsed(seq int) int {
	var used int
	row := mysql.Cli.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM voucher WHERE seq = %d", seq))
	row.Scan(&used)
	return used
}