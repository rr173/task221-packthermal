// 冷链包装热阻参数反演服务入口。
//
// 用法：
//
//	packthermal --addr :8080 --db packthermal.db   # 启动 HTTP 服务
//	packthermal --smoke-test                       # 确定性端到端自检后退出（Docker 判据）
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"task221-packthermal/internal/httpapi"
	"task221-packthermal/internal/service"
	"task221-packthermal/internal/store"
)

func main() {
	dbPath := flag.String("db", "packthermal.db", "SQLite database path")
	address := flag.String("addr", ":8080", "HTTP listen address")
	smoke := flag.Bool("smoke-test", false, "run deterministic end-to-end self-check and exit")
	flag.Parse()

	if *smoke {
		runSmoke(*dbPath)
		return
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	repos := store.NewRepositories(st)
	app := service.New(repos)

	srv := &http.Server{
		Addr:              *address,
		Handler:           httpapi.New(app).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("task221-packthermal listening on %s (db=%s)", *address, *dbPath)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// runSmoke 执行确定性端到端自检：从空库演示完整闭环，关闭重开同一数据库
// 验证持久化与重启恢复，最后清理临时库并以 0 退出。
func runSmoke(dbPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDB := dbPath + ".smoke-" + fmt.Sprintf("%d", os.Getpid()) + ".db"
	st, err := store.Open(tmpDB)
	if err != nil {
		log.Fatalf("smoke open store: %v", err)
	}
	app := service.New(store.NewRepositories(st))
	if err := app.RunDemo(ctx); err != nil {
		st.Close()
		os.Remove(tmpDB)
		log.Fatalf("smoke test failed: %v", err)
	}
	if err := st.Close(); err != nil {
		log.Fatalf("smoke close store: %v", err)
	}

	// 关闭后重开同一数据库，验证持久化与重启恢复。
	stReopen, err := store.Open(tmpDB)
	if err != nil {
		log.Fatalf("smoke reopen store: %v", err)
	}
	appReopen := service.New(store.NewRepositories(stReopen))
	trials, err := appReopen.ListTrials(ctx)
	if err != nil || len(trials) == 0 {
		log.Fatalf("smoke restart check failed: trials=%d err=%v", len(trials), err)
	}
	series, err := appReopen.ListSeries(ctx, trials[0].ID)
	if err != nil || len(series) == 0 {
		log.Fatalf("smoke restart check failed: series=%d err=%v", len(series), err)
	}
	snaps, err := appReopen.ListSnapshots(ctx, trials[0].ID)
	if err != nil || len(snaps) == 0 {
		log.Fatalf("smoke restart check failed: snapshots=%d err=%v", len(snaps), err)
	}
	invs, err := appReopen.ListInversions(ctx, trials[0].ID)
	if err != nil || len(invs) == 0 {
		log.Fatalf("smoke restart check failed: inversions=%d err=%v", len(invs), err)
	}
	check, err := appReopen.SelfCheck(ctx)
	if err != nil || check["integrity_check"] != 1 {
		log.Fatalf("smoke selfcheck failed: %+v err=%v", check, err)
	}
	stReopen.Close()
	os.Remove(tmpDB)
	fmt.Fprintln(os.Stdout, "task221-packthermal smoke test passed")
}
