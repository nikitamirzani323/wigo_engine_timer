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
	fmt.Printf("Create First new invoice %s\n", invoice)
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
	// for sec >= 0 {
	// 	time.Sleep(1 * time.Second)
	// 	sec--
	// }
	invoice = ""
	time_status = "LOCK"
	data_send = invoice + "|0|" + strconv.Itoa(sec%60) + "|" + time_status
	fmt.Printf("%s:%.2d:%.2d:%s\r", invoice, 0, sec%60, time_status)
	senddata(data_send)

	flag_compile = Update_transaksi("nuke")
	fmt.Printf("status compile %t\n", flag_compile)
	if flag_compile {
		invoice = Save_transaksi("nuke", "IDR")
		fmt.Printf("Create new invoice %s\n", invoice)
	}
	fmt.Printf("status compile 2 %t\n", flag_compile)
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
				idtransaksi , idcurr, idcompany, datetransaksi, status_transaksi, 
				create_transaksi, createdate_transaksi 
			) values (
				$1, $2, $3, $4, $5,  
				$6, $7  
			)
		`

		field_column := tbl_trx_transaksi + tglnow.Format("YYYY") + tglnow.Format("MM")
		idrecord_counter := models.Get_counter(field_column)
		idrecrodparent_value := tglnow.Format("YY") + tglnow.Format("MM") + tglnow.Format("DD") + tglnow.Format("HH") + strconv.Itoa(idrecord_counter)
		date_transaksi := tglnow.Format("YYYY-MM-DD HH:mm:ss")

		flag_insert, msg_insert := models.Exec_SQL(sql_insert, tbl_trx_transaksi, "INSERT",
			idrecrodparent_value, idcurr, idcompany, date_transaksi, "OPEN",
			"SYSTEM", date_transaksi)

		if flag_insert {
			id_invoice = idrecrodparent_value

		} else {
			fmt.Println(msg_insert)
		}
	}

	return id_invoice
}
func Update_transaksi(idcompany string) bool {
	flag_compileupdate := false
	tglnow, _ := goment.New()
	id_invoice := _GetInvoice(idcompany)
	prize_2D := helpers.GenerateNumber(2)

	if id_invoice != "" {
		_, tbl_trx_transaksi, tbl_trx_transaksidetail, _ := models.Get_mappingdatabase(idcompany)
		// UPDATE RESULT PARENT
		sql_update := `
				UPDATE 
				` + tbl_trx_transaksi + `  
				SET resultwigo=$1, status_transaksi=$2, 
				update_transaksi=$3, updatedate_transaksi=$4           
				WHERE idtransaksi=$5          
			`

		flag_update, msg_update := models.Exec_SQL(sql_update, tbl_trx_transaksi, "UPDATE",
			prize_2D, "CLOSED",
			"SYSTEM", tglnow.Format("YYYY-MM-DD HH:mm:ss"), id_invoice)

		if flag_update {
			con := db.CreateCon()
			ctx := context.Background()
			flag_detail := false
			sql_select_detail := `SELECT 
					idtransaksidetail , nomor, bet, multiplier
					FROM ` + tbl_trx_transaksidetail + `  
					WHERE status_transaksidetail='RUNNING'  
					AND idtransaksi='` + id_invoice + `'  `

			row, err := con.QueryContext(ctx, sql_select_detail)
			helpers.ErrorCheck(err)
			for row.Next() {
				var (
					bet_db                         int
					multiplier_db                  float64
					idtransaksidetail_db, nomor_db string
				)

				err = row.Scan(&idtransaksidetail_db, &nomor_db, &bet_db, &multiplier_db)
				helpers.ErrorCheck(err)

				status_client := _rumuswigo(nomor_db, prize_2D)
				win := 0
				if status_client == "WIN" {
					win = bet_db + int(float64(bet_db)*multiplier_db)
				}

				// UPDATE STATUS DETAIL
				sql_update_detail := `
					UPDATE 
					` + tbl_trx_transaksidetail + `  
					SET status_transaksidetail=$1, win=$2, 
					update_transaksidetail=$3, updatedate_transaksidetail=$4           
					WHERE idtransaksidetail=$5          
				`
				flag_update_detail, msg_update_detail := models.Exec_SQL(sql_update_detail, tbl_trx_transaksidetail, "UPDATE",
					status_client, win,
					"SYSTEM", tglnow.Format("YYYY-MM-DD HH:mm:ss"), idtransaksidetail_db)

				if !flag_update_detail {
					fmt.Println(msg_update_detail)
				}
				flag_detail = true
			}
			defer row.Close()
			if flag_detail {
				// UPDATE PARENT
				total_bet, total_win := _GetTotalBetWin_Transaksi(tbl_trx_transaksidetail, id_invoice)
				sql_update_parent := `
					UPDATE 
					` + tbl_trx_transaksi + `  
					SET total_bet=$1, total_win=$2, 
					update_transaksi=$3, updatedate_transaksi=$4           
					WHERE idtransaksi=$5       
				`
				flag_update_parent, msg_update_parent := models.Exec_SQL(sql_update_parent, tbl_trx_transaksi, "UPDATE",
					total_bet, total_win,
					"SYSTEM", tglnow.Format("YYYY-MM-DD HH:mm:ss"), id_invoice)

				if !flag_update_parent {
					fmt.Println(msg_update_parent)
				} else {
					flag_compileupdate = true
				}
			} else {
				flag_compileupdate = true
			}

		} else {
			fmt.Println(msg_update)
		}
	}

	return flag_compileupdate
}
func senddata(data string) {
	helpers.SetPublish("payload", data)
}
func _GetInvoice(idcompany string) string {
	con := db.CreateCon()
	ctx := context.Background()

	_, tbl_trx_transaksi, _, _ := models.Get_mappingdatabase(idcompany)

	idtransaksi := ""

	sql_select := ""
	sql_select += "SELECT "
	sql_select += "idtransaksi "
	sql_select += "FROM " + tbl_trx_transaksi + " "
	sql_select += "WHERE resultwigo='' ORDER BY idtransaksi DESC LIMIT 1"

	row := con.QueryRowContext(ctx, sql_select)
	switch e := row.Scan(&idtransaksi); e {
	case sql.ErrNoRows:
	case nil:
	default:
		helpers.ErrorCheck(e)
	}

	return idtransaksi
}
func _GetTotalBetWin_Transaksi(table, idtransaksi string) (int, int) {
	con := db.CreateCon()
	ctx := context.Background()
	total_bet := 0
	total_win := 0
	sql_select := ""
	sql_select += "SELECT "
	sql_select += "SUM(bet) AS total_bet, SUM(win) AS total_win  "
	sql_select += "FROM " + table + " "
	sql_select += "WHERE idtransaksi='" + idtransaksi + "'   "
	sql_select += "AND status_transaksidetail !='RUNNING'   "

	row := con.QueryRowContext(ctx, sql_select)
	switch e := row.Scan(&total_bet, &total_win); e {
	case sql.ErrNoRows:
	case nil:
	default:
		helpers.ErrorCheck(e)
	}

	return total_bet, total_win
}
func _rumuswigo(nomorclient, nomorkeluaran string) string {
	result := "LOSE"
	if nomorclient == nomorkeluaran {
		result = "WIN"
	}
	return result
}
