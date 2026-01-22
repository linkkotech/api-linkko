#!/usr/bin/env bash
# Script para gerar JWT_HS256_SECRET válido
# Uso: ./scripts/generate-jwt-secret.sh

set -e

echo "🔐 Generating JWT_HS256_SECRET for HMAC SHA-256..."
echo ""

# Gera 32 bytes aleatórios e codifica em Base64
JWT_SECRET=$(openssl rand -base64 32)

echo "✅ Generated Base64-encoded secret (32 bytes):"
echo "$JWT_SECRET"
echo ""

# Validar tamanho após decodificar
DECODED_SIZE=$(echo "$JWT_SECRET" | base64 -d | wc -c)
echo "📊 Decoded size: $DECODED_SIZE bytes (minimum: 32 bytes for HS256)"
echo ""

if [ "$DECODED_SIZE" -lt 32 ]; then
    echo "❌ WARNING: Secret too short! Must be at least 32 bytes."
    exit 1
fi

echo "✅ Secret is valid for HS256 (256-bit HMAC)"
echo ""
echo "📝 Add to your .env file:"
echo "JWT_HS256_SECRET=$JWT_SECRET"
echo ""
echo "💡 Test decoding:"
echo "echo '$JWT_SECRET' | base64 -d | wc -c"
