//go:build !windows

package service

import "github.com/xyztony999/devswitch/internal/models"

func writeUserEnvPlatform(*models.State) {}
