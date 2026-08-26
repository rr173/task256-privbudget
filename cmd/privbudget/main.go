// 命令 privbudget 是差分隐私统计预算组合验证服务的入口。
//
// 启动模式：
//
//	go run ./cmd/privbudget --addr :8080 --db privbudget.db
//
// 自检模式（不启动长驻服务，验证持久化与重启恢复后以 0 退出）：
//
//	go run ./cmd/privbudget --smoke-test
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"task256-privbudget/internal/httpapi"
	"task256-privbudget/internal/service"
	"task256-privbudget/internal/store"
)

func main() {
	var addr, dbPath string
	var smoke bool
	flag.StringVar(&addr, "addr", ":8080", "HTTP 监听地址")
	flag.StringVar(&dbPath, "db", "privbudget.db", "SQLite 数据库路径")
	flag.BoolVar(&smoke, "smoke-test", false, "运行自检后退出（不启动长驻服务）")
	flag.Parse()

	if smoke {
		if err := runSmoke(dbPath); err != nil {
			fmt.Fprintln(os.Stderr, "smoke test FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("smoke test passed")
		os.Exit(0)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()

	app := service.NewApp(st)
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewHandler(app).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("privbudget listening on %s (db=%s)", addr, dbPath)
	log.Fatal(srv.ListenAndServe())
}
