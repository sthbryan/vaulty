package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/storage"
	authusecase "github.com/DeadBryam/vaulty/pkg/application/usecases/auth"
)

func ensureAuthenticated(cfg *config.Config) (*session.Session, error) {
	if cfg.Repo == "" && !cfg.IsLocalMode() {
		return nil, fmt.Errorf("vaulty not initialized. Run 'vty init' first")
	}

	if cfg.CurrentUser == "" {
		return nil, fmt.Errorf("no current user set. Run 'vty login' first")
	}

	sessionManager := session.GetManager()
	existingSession := sessionManager.Get(cfg.CurrentUser)
	if existingSession != nil && existingSession.IsActive() {
		return existingSession, nil
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return nil, fmt.Errorf("password storage: %w", err)
	}

	storedPassword, err := passStorage.Get()
	if err != nil {
		return nil, fmt.Errorf("no active session. Run 'vty login' first")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	factory := storage.NewFactory(cfg)
	loginUseCase := authusecase.NewLoginUseCase(factory)

	output, err := loginUseCase.Execute(ctx, authusecase.LoginInput{
		Username:       cfg.CurrentUser,
		MasterPassword: storedPassword,
	})
	if err != nil {
		return nil, err
	}

	sessionManager.Create(output.Session)

	return output.Session, nil
}
