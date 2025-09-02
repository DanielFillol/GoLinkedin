#!/bin/bash

# Script wrapper para executar o crawler suprimindo erros de DevTools
# Uso: ./run_crawler.sh [argumentos do crawler]

# Executar o crawler e filtrar apenas os erros específicos do Chrome DevTools
go run ./cmd/crawler/main.go "$@" 2>&1 | grep -v "could not unmarshal event" | grep -v "parse error" | grep -v "cookiePart" | grep -v "ClientNavigationReason"
