// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Chris <goabonga@pm.me>

// Command infra-idp is a placeholder entry point scaffolded in M0. The real
// implementation lands in later milestones (see PLAN.md).
package main

import (
	"fmt"

	"github.com/goabonga/infrastructure/internal/meta"
)

func main() {
	fmt.Println(meta.Line("infra-idp", Version))
}
