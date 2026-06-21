package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/1v4mp1r3/light-log-aggregator/internal/server"
	"github.com/1v4mp1r3/light-log-aggregator/internal/store"
	"github.com/1v4mp1r3/light-log-aggregator/internal/syslog"
)

const version = "v0.1.0"

func main() {
	var listen string
	var syslogListen string
	var storePath string
	var retention time.Duration
	var showVersion bool

	flag.StringVar(&listen, "listen", ":8080", "HTTP listen address.")
	flag.StringVar(&syslogListen, "syslog", "", "Optional UDP syslog listen address, e.g. :5514.")
	flag.StringVar(&storePath, "store", "data/loglite.jsonl", "Append-only JSONL store path. Empty keeps memory only.")
	flag.DurationVar(&retention, "retention", 7*24*time.Hour, "Retention window. Use 0 to disable pruning.")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit.")
	flag.Parse()

	if showVersion {
		fmt.Println("loglite", version)
		return
	}

	st, err := store.New(storePath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	app := server.New(st, version)

	if removed, err := st.Retain(retention); err != nil {
		log.Fatalf("retention: %v", err)
	} else if removed > 0 {
		log.Printf("retention removed %d entries", removed)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if syslogListen != "" {
		go runSyslog(ctx, syslogListen, app)
	}
	go runRetention(ctx, st, retention)

	httpServer := &http.Server{
		Addr:              listen,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("loglite %s listening on %s", version, listen)
	if syslogListen != "" {
		log.Printf("udp syslog listening on %s", syslogListen)
	}
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server: %v", err)
	}
}

func runRetention(ctx context.Context, st *store.Store, window time.Duration) {
	if window <= 0 {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if removed, err := st.Retain(window); err != nil {
				log.Printf("retention failed: %v", err)
			} else if removed > 0 {
				log.Printf("retention removed %d entries", removed)
			}
		}
	}
}

func runSyslog(ctx context.Context, addr string, app *server.Server) {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Printf("syslog listener failed: %v", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 8192)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			log.Printf("syslog read failed: %v", err)
			continue
		}

		entry := syslog.Parse(string(buf[:n]), time.Now())
		if _, err := app.Add(entry); err != nil {
			log.Printf("syslog ingest failed: %v", err)
		}
	}
}
