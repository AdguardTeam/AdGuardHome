package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQLドライバ
)

type LicenseInfo struct {
	Key        string    `json:"key"`
	Expiration time.Time `json:"expiration"`
}

func GetLicenseInfo(w http.ResponseWriter, r *http.Request) {
	// ライセンスキーをテキストファイルから読み取る
	key, err := os.ReadFile("/home/tukimoto/AdGuardHome/internal/config/license_key.txt")
	if err != nil {
		http.Error(w, "ライセンスファイルを読み込めませんでした", http.StatusInternalServerError)
		return
	}

	// MySQLに接続
	db, err := sql.Open("mysql", "root:mss0804mss@tcp(localhost:3306)/licenses")
	if err != nil {
		http.Error(w, "データベースに接続できませんでした", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var expiration time.Time
	query := "SELECT expiration_date FROM licenses WHERE license_key = ?"
	err = db.QueryRow(query, string(key)).Scan(&expiration)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "ライセンスキーが見つかりませんでした", http.StatusNotFound)
		} else {
			http.Error(w, "データベースエラーが発生しました", http.StatusInternalServerError)
		}
		return
	}

	licenseInfo := LicenseInfo{
		Key:        string(key),
		Expiration: expiration,
	}
	json.NewEncoder(w).Encode(licenseInfo)
}
