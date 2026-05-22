// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

package state

import "context"

// Open returns the configured store: a PostgreSQL-backed store when dsn is set
// (highly available, multi-instance), otherwise a file store rooted at dir.
func Open(dir, dsn string) (Store, error) {
	if dsn != "" {
		return NewPostgresStore(context.Background(), dsn)
	}
	return NewFileStore(dir), nil
}
