# Task 002 – Conversation & Message Service

## Goal

Enable 1-1 and group chat with persistent messages.

## Features

- [ ] Model: Conversation (id, type, name, created_at)
- [ ] Model: ConversationMember (conversation_id, user_id, role)
- [ ] Model: Message (conversation_id, sender_id, content)
- [ ] API: Create conversation (POST /conversations)
- [ ] API: List conversations (GET /conversations)
- [ ] API: Send message (POST /conversations/:id/messages)
- [ ] API: List messages with pagination
- [ ] API: Edit and delete message
- [ ] Repository layer for conversations and messages
- [ ] Service layer (CreateMessage, GetMessages)
- [ ] Add foreign keys + migrations
