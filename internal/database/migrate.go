package database

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) error {
	// golang-migrate com driver pgx/v5 requer o scheme "pgx5://"
	migrateURL := strings.NewReplacer(
		"postgresql://", "pgx5://",
		"postgres://", "pgx5://",
	).Replace(databaseURL)

	m, err := migrate.New("file://migrations", migrateURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}

		// Se o banco ficou em estado dirty (migration interrompida),
		// forca a versao anterior e tenta novamente.
		var dirtyErr *migrate.ErrDirty
		if errors.As(err, &dirtyErr) {
			if forceErr := m.Force(dirtyErr.Version - 1); forceErr != nil {
				return fmt.Errorf("migration failed (dirty v%d), force also failed: %w", dirtyErr.Version, forceErr)
			}
			if upErr := m.Up(); upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
				return fmt.Errorf("migration failed after dirty recovery: %w", upErr)
			}
			return nil
		}

		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
