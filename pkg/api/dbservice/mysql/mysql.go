package mysql

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strings"
	"time"
	"voucher/pkg/config"
)

var Cli *sql.DB

func InitMysqlClient(conf *config.AppConfig) {
	user := conf.App.Mysql.User
	pwd := conf.App.Mysql.Password
	addr := conf.App.Mysql.Address
	dbName := conf.App.Mysql.Dbname
	dbUrl := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		user, pwd, addr, dbName)

	db, err := sql.Open("mysql", dbUrl)
	if err != nil {
		log.Fatalln(err)
	}
	Cli = db
}

type Voucher struct {
	Seq int
	Code string
	Used bool
}

func GetUserSeqVoucher(usr string, seqs ...int) []Voucher {
	if len(seqs) == 0 {
		return nil
	}
	stmt := fmt.Sprintf(`SELECT seq, code, used FROM voucher WHERE usr = %s AND seq IN (?,` + strings.Repeat(",?", len(seqs) - 1) + `)`, usr)
	rows, _ := Cli.Query(stmt, seqs)
	res := make([]Voucher, 0)
	for rows.Next() {
		var seq int
		var code string
		var used bool
		rows.Scan(&seq, &code, &used)
		res = append(res, Voucher{seq, code, used})
	}
	return res
}

type Activity struct {
	Seq int
	StartTime time.Time
	EndTime time.Time
	RedeemLimit int
	Rest int
}

func GetActivities() []Activity {
	stmt := fmt.Sprintf("SELECT seq, startTime, endTime, redeemLimit, rest FROM activity")
	rows, _ := Cli.Query(stmt)
	res := make([]Activity, 0)
	for rows.Next() {
		tmp := Activity{}
		rows.Scan(&tmp.Seq, &tmp.StartTime, &tmp.EndTime, &tmp.RedeemLimit, &tmp.Rest)
		res = append(res, tmp)
	}
	return res
}

func SetAllocatedVoucher(usr string, code string, seq int) error {
	stmt := fmt.Sprintf("UPDATE voucher SET usr=%s WHERE code=%s", usr, code)
	_, err := Cli.Exec(stmt)
	if err != nil {
		return err
	}
	stmt = fmt.Sprintf("UPDATE activity SET rest = rest - 1 WHERE seq = %d", seq)
	_, err = Cli.Exec(stmt)
	return err
}

func SetUsedVoucher(code string) error {
	stmt := fmt.Sprintf("UPDATE voucher SET used = true WHERE code=%s", code)
	_, err := Cli.Exec(stmt)
	return err
}

func GetAliveUnallocatedVouchers(aliveSeqs []interface{}, limit int) []Voucher {
	if len(aliveSeqs) == 0 {
		return nil
	}
	stmt := fmt.Sprintf(`SELECT seq, code FROM voucher WHERE usr = "" AND seq IN (?` + strings.Repeat(",?", len(aliveSeqs)-1) + `) ORDER BY seq DESC`)
	if limit >= 0 {
		stmt += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := Cli.Query(stmt, aliveSeqs...)
	res := make([]Voucher, 0)
	for rows.Next() {
		var seq int
		var code string
		err = rows.Scan(&seq, &code)
		if err != nil {
			break
		}
		res = append(res, Voucher{Code: code, Seq: seq, Used: false})
	}
	return res
}