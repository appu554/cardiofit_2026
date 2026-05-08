// Command kb-30-authorisation-evaluator is the runtime authorisation
// evaluator service. Listens on port 8138 by default.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"kb-authorisation-evaluator/internal/api"
	"kb-authorisation-evaluator/internal/audit"
	"kb-authorisation-evaluator/internal/cache"
	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
	"kb-authorisation-evaluator/internal/invalidation"
	credentialresolver "kb-authorisation-evaluator/internal/resolver"
	"kb-authorisation-evaluator/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8138"
	}

	var (
		s        store.Store
		resolver evaluator.ConditionResolver
	)
	if dsn := os.Getenv("KB30_DATABASE_URL"); dsn != "" {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("open kb30 postgres: %v", err)
		}
		if err := db.Ping(); err != nil {
			log.Fatalf("ping kb30 postgres: %v", err)
		}
		log.Printf("kb-30: using PostgresStore (KB30_DATABASE_URL set)")
		s = store.NewPostgresStore(db)
		resolver = credentialresolver.NewCredentialResolver(db)
		log.Printf("kb-30: using CredentialResolver (real)")
	} else {
		log.Printf("kb-30: using MemoryStore (KB30_DATABASE_URL unset)")
		s = store.NewMemoryStore()
		resolver = evaluator.AlwaysPassResolver
		log.Printf("kb-30: using AlwaysPassResolver (test)")
	}
	loadExamples(s)

	var c cache.Cache
	if redisAddr := os.Getenv("KB30_REDIS_ADDR"); redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			log.Fatalf("ping kb30 redis at %s: %v", redisAddr, err)
		}
		log.Printf("kb-30: using RedisCache at %s", redisAddr)
		c = cache.NewRedisFromClient(rdb)
	} else {
		log.Printf("kb-30: using InMemoryCache (KB30_REDIS_ADDR unset)")
		c = cache.NewInMemory()
	}
	auditSvc := audit.NewService()
	eval := evaluator.New(s, resolver)

	server := &api.Server{Evaluator: eval, Cache: c, Audit: auditSvc}

	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("kb-30-authorisation-evaluator listening on :%s", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Kafka consumer for substrate-driven cache invalidation. Started as a
	// goroutine so server boot is not blocked by Kafka availability; the
	// consumer logs read errors and continues, never crashing main.
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()
	if brokers := os.Getenv("KB30_KAFKA_BROKERS"); brokers != "" {
		topic := os.Getenv("KB30_KAFKA_TOPIC")
		if topic == "" {
			topic = "substrate_updates"
		}
		kc := &invalidation.KafkaConsumer{
			Brokers: strings.Split(brokers, ","),
			Topic:   topic,
			Inv:     invalidation.New(c),
		}
		go func() {
			if err := kc.Run(consumerCtx); err != nil &&
				!errors.Is(err, context.Canceled) {
				log.Printf("kb-30 kafka consumer exited: %v", err)
			}
		}()
		log.Printf("kb-30: started Kafka consumer for topic %s on %s", topic, brokers)
	} else {
		log.Printf("kb-30: KB30_KAFKA_BROKERS unset; skipping Kafka consumer (cache invalidation only via direct API)")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancelConsumer()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	log.Print("shutdown complete")
}

// loadExamples ingests the bundled example rules into the in-memory store.
// Production wiring would use the PostgresStore + a migration-driven seed.
func loadExamples(s store.Store) {
	dir := "examples"
	if env := os.Getenv("KB30_EXAMPLES_DIR"); env != "" {
		dir = env
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("loadExamples: skipping (%v)", err)
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			log.Printf("loadExamples: read %s: %v", e.Name(), err)
			continue
		}
		rule, err := dsl.ParseRule(data)
		if err != nil {
			log.Printf("loadExamples: parse %s: %v", e.Name(), err)
			continue
		}
		if _, err := s.Insert(context.Background(), *rule, data); err != nil {
			log.Printf("loadExamples: insert %s: %v", e.Name(), err)
			continue
		}
		log.Printf("loaded example rule %s", rule.RuleID)
	}
}
