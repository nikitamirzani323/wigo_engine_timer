package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/nikitamirzani323/wigo_engine_timer/db"
	"github.com/nikitamirzani323/wigo_engine_timer/helpers"
	"github.com/nikitamirzani323/wigo_engine_timer/models"
	"github.com/nleeper/goment"
)

var invoice = ""
var time_status = "LOCK"
var data_send = ""

func main() {
	time_game := 30
	time_compile := 20
	flag := true

	err := godotenv.Load()
	if err != nil {
		panic("Failed to load env file")
	}

	initRedis := helpers.RedisHealth()

	if !initRedis {
		panic("cannot load redis")
	}

	db.Init()
	invoice = Save_transaksi("nuke", "IDR")

	for flag {
		flag = loop_count(time_game, time_compile)
	}

}
func loop_count(sec, compile int) bool {
	flag := false
	fmt.Println("")
	for sec >= 0 {
		time_status = "OPEN"
		data_send = invoice + "|" + strconv.Itoa(sec%60) + "|0|" + time_status
		fmt.Printf("%s:%.2d:%.2d:%s\r", invoice, sec%60, 0, time_status)
		senddata(data_send)
		time.Sleep(1 * time.Second)
		sec--
	}
	flag = loop_compile(compile)
	return flag
}
func loop_compile(sec int) bool {
	flag_compile := false
	fmt.Println("")
	for sec >= 0 {
		time_status = "LOCK"
		data_send = invoice + "|0|" + strconv.Itoa(sec%60) + "|" + time_status
		fmt.Printf("%s:%.2d:%.2d:%s\r", invoice, 0, sec%60, time_status)
		senddata(data_send)
		// fmt.Printf("COMPILE %.2d\r", sec%60)
		time.Sleep(1 * time.Second)
		sec--
	}
	invoice = Save_transaksi("nuke", "IDR")
	flag_compile = true
	return flag_compile
}
func Save_transaksi(idcompany, idcurr string) string {
	tglnow, _ := goment.New()
	id_invoice := _GetInvoice(idcompany)
	if id_invoice == "" {
		_, tbl_trx_transaksi, _, _ := models.Get_mappingdatabase(idcompany)
		sql_insert := `
			insert into
			` + tbl_trx_transaksi + ` (
				idtransaksi , idcurr, idcompany, datetransaksi,
				create_transaksi, createdate_transaksi 
			) values (
				$1, $2, $3, $4, 
				$5, $6 
			)
		`

		field_column := tbl_trx_transaksi + tglnow.Format("YYYY") + tglnow.Format("MM")
		idrecord_counter := models.Get_counter(field_column)
		idrecrodparent_value := tglnow.Format("YY") + tglnow.Format("MM") + tglnow.Format("DD") + tglnow.Format("HH") + strconv.Itoa(idrecord_counter)
		date_transaksi := tglnow.Format("YYYY-MM-DD HH:mm:ss")

		flag_insert, msg_insert := models.Exec_SQL(sql_insert, tbl_trx_transaksi, "INSERT",
			idrecrodparent_value, idcurr, idcompany, date_transaksi,
			"SYSTEM", date_transaksi)

		if flag_insert {
			id_invoice = idrecrodparent_value

		} else {
			fmt.Println(msg_insert)
		}
	}

	return id_invoice
}
func senddata(data string) {
	helpers.SetPublish("payload", data)
}
func _GetInvoice(idcompany string) string {
	con := db.CreateCon()
	ctx := context.Background()

	_, tbl_trx_transaksi, _, _ := models.Get_mappingdatabase(idcompany)

	resultwigo := ""

	sql_select := ""
	sql_select += "SELECT "
	sql_select += "idtransaksi "
	sql_select += "FROM " + tbl_trx_transaksi + " "
	sql_select += "WHERE resultwigo='' LIMIT 1"

	row := con.QueryRowContext(ctx, sql_select)
	switch e := row.Scan(&resultwigo); e {
	case sql.ErrNoRows:
	case nil:
	default:
		helpers.ErrorCheck(e)
	}

	return resultwigo
}
