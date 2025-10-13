#!/bin/bash
# ChatterGo API Usage Examples
# This script demonstrates how to use the ChatterGo API

BASE_URL="http://localhost:8080/api/v1"

echo "=== ChatterGo API Usage Examples ==="
echo ""

# 1. Register a new user
echo "1. Registering a new user..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "securepassword123"
  }')

echo "$REGISTER_RESPONSE" | jq '.'
ACCESS_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.access_token')
echo "Access Token: $ACCESS_TOKEN"
echo ""

# 2. Login
echo "2. Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword123"
  }')

echo "$LOGIN_RESPONSE" | jq '.'
ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')
echo ""

# 3. Create a room
echo "3. Creating a chat room..."
ROOM_RESPONSE=$(curl -s -X POST "$BASE_URL/rooms" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "name": "General Discussion",
    "description": "A place for general chat"
  }')

echo "$ROOM_RESPONSE" | jq '.'
ROOM_ID=$(echo "$ROOM_RESPONSE" | jq -r '.id')
echo "Room ID: $ROOM_ID"
echo ""

# 4. List all rooms
echo "4. Listing all rooms..."
curl -s -X GET "$BASE_URL/rooms" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'
echo ""

# 5. Join a room (if not already joined)
echo "5. Joining the room..."
curl -s -X POST "$BASE_URL/rooms/join" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{
    \"room_id\": $ROOM_ID
  }" | jq '.'
echo ""

# 6. Send a message
echo "6. Sending a message..."
MESSAGE_RESPONSE=$(curl -s -X POST "$BASE_URL/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{
    \"room_id\": $ROOM_ID,
    \"content\": \"Hello, everyone! This is my first message.\"
  }")

echo "$MESSAGE_RESPONSE" | jq '.'
echo ""

# 7. Get message history
echo "7. Getting message history..."
curl -s -X GET "$BASE_URL/messages/history?room_id=$ROOM_ID&limit=10&offset=0" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'
echo ""

# 8. Get room members
echo "8. Getting room members..."
curl -s -X GET "$BASE_URL/rooms/$ROOM_ID/members" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'
echo ""

echo "=== Done! ==="
echo ""
echo "To connect via WebSocket:"
echo "ws://localhost:8080/api/v1/ws?room_id=$ROOM_ID"
echo "Authorization: Bearer $ACCESS_TOKEN"
