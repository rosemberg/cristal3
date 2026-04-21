#!/bin/bash
# Simple test script for cristal-backend API

set -e

API_URL="http://localhost:8080"

echo "🧪 Testing Cristal Backend API"
echo "================================"
echo

# Health check
echo "1. Health Check"
echo "   GET $API_URL/health"
response=$(curl -s $API_URL/health)
echo "   Response: $response"
if echo "$response" | grep -q '"status":"ok"'; then
    echo "   ✓ Health check passed"
else
    echo "   ✗ Health check failed"
    exit 1
fi
echo

# Chat request - Simple query
echo "2. Chat Request - Simple Query"
echo "   POST $API_URL/chat"
request='{"message": "O que é o catálogo?"}'
echo "   Request: $request"
response=$(curl -s -X POST $API_URL/chat \
    -H "Content-Type: application/json" \
    -d "$request")
echo "   Response: $response"
if echo "$response" | grep -q '"status":"success"'; then
    echo "   ✓ Chat request successful"
else
    echo "   ✗ Chat request failed"
    echo "   Response: $response"
    exit 1
fi
echo

# Chat request - Search query
echo "3. Chat Request - Search Query"
echo "   POST $API_URL/chat"
request='{"message": "Busque páginas sobre balancetes"}'
echo "   Request: $request"
response=$(curl -s -X POST $API_URL/chat \
    -H "Content-Type: application/json" \
    -d "$request")
echo "   Response: $response"
if echo "$response" | grep -q '"status":"success"'; then
    echo "   ✓ Search query successful"
else
    echo "   ✗ Search query failed"
    exit 1
fi
echo

echo "================================"
echo "✓ All tests passed!"
