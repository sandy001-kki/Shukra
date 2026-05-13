package cleanup

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	_ "github.com/lib/pq"
)

type DatabaseHook struct {
	client client.Client
}

func NewDatabaseHook(c client.Client) Hook {
	return &DatabaseHook{client: c}
}

func (h *DatabaseHook) Name() string {
	return "database-schema-cleanup"
}

func (h *DatabaseHook) Cleanup(ctx context.Context, env CleanupTarget) error {
	if env.DatabaseSecret == "" {
		ctrl.LoggerFrom(ctx).Info("database cleanup skipped; no database secret configured")
		return nil
	}

	hookCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	log := ctrl.LoggerFrom(ctx).WithValues("hook", h.Name(), "databaseMode", env.DatabaseMode, "secret", env.DatabaseSecret)
	secret := &corev1.Secret{}
	if err := h.client.Get(hookCtx, types.NamespacedName{Name: env.DatabaseSecret, Namespace: env.Namespace}, secret); err != nil {
		log.Error(err, "database secret unavailable; treating cleanup as best-effort")
		return nil
	}

	dsn := databaseDSN(secret)
	if dsn == "" {
		log.Info("database cleanup skipped; secret does not contain PostgreSQL connection details")
		return nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error(err, "database cleanup could not initialize PostgreSQL client")
		return nil
	}
	defer db.Close()

	if err := db.PingContext(hookCtx); err != nil {
		log.Error(err, "database unreachable; treating cleanup as best-effort")
		return nil
	}

	if _, err := db.ExecContext(hookCtx, `DROP TABLE IF EXISTS shukra_schema_migrations`); err != nil {
		return fmt.Errorf("%w: drop migration metadata: %v", ErrTransient, err)
	}
	if _, err := db.ExecContext(hookCtx, `
CREATE TABLE IF NOT EXISTS shukra_schema_metadata (
	name text PRIMARY KEY,
	namespace text NOT NULL,
	decommissioned_at timestamptz NOT NULL DEFAULT now()
)`); err != nil {
		return fmt.Errorf("%w: ensure schema metadata table: %v", ErrTransient, err)
	}
	if _, err := db.ExecContext(hookCtx, `
INSERT INTO shukra_schema_metadata (name, namespace, decommissioned_at)
VALUES ($1, $2, now())
ON CONFLICT (name) DO UPDATE SET namespace = EXCLUDED.namespace, decommissioned_at = now()`, env.Name, env.Namespace); err != nil {
		return fmt.Errorf("%w: mark schema decommissioned: %v", ErrTransient, err)
	}

	log.Info("database schema metadata cleanup completed")
	return nil
}

func databaseDSN(secret *corev1.Secret) string {
	if dsn := firstSecretValue(secret, "dsn", "databaseURL", "database_url", "DATABASE_URL"); dsn != "" {
		return dsn
	}

	host := firstSecretValue(secret, "host", "hostname", "POSTGRES_HOST")
	user := firstSecretValue(secret, "user", "username", "POSTGRES_USER")
	password := firstSecretValue(secret, "password", "POSTGRES_PASSWORD")
	dbname := firstSecretValue(secret, "dbname", "database", "POSTGRES_DB")
	if host == "" || user == "" || dbname == "" {
		return ""
	}

	values := url.Values{}
	values.Set("sslmode", firstNonEmpty(firstSecretValue(secret, "sslmode", "POSTGRES_SSLMODE"), "require"))
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?%s", url.QueryEscape(user), url.QueryEscape(password), host, dbname, values.Encode())
	return dsn
}

func firstSecretValue(secret *corev1.Secret, keys ...string) string {
	for _, key := range keys {
		if val, ok := secret.Data[key]; ok {
			return strings.TrimSpace(string(val))
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
