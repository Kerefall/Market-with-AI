package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	CORSOrigin  string
	// ИИ: любой OpenAI-совместимый API. Если LLM_API_URL пуст, работает демо-режим (шаблоны).
	LLMURL   string
	LLMKey   string
	LLMModel string
	// CloudPayments (платёжный виджет). Если пусто, остаётся тестовая оплата без реальных денег.
	CPPublicID  string
	CPAPISecret string
}

func Load() (*Config, error) {
	c := &Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		CORSOrigin:  getenv("CORS_ORIGIN", "*"),
		LLMURL:      os.Getenv("LLM_API_URL"),
		LLMKey:      os.Getenv("LLM_API_KEY"),
		LLMModel:    getenv("LLM_MODEL", "gpt-4o-mini"),

		CPPublicID:  os.Getenv("CLOUDPAYMENTS_PUBLIC_ID"),
		CPAPISecret: os.Getenv("CLOUDPAYMENTS_API_SECRET"),
	}
	if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	if len(c.JWTSecret) < 16 {
		return nil, errors.New("JWT_SECRET must be at least 16 characters")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
