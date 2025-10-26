package models

// All returns a slice of pointers to all GORM models in the project.
// Keep this list updated when adding new models to ensure AutoMigrate and tooling see them.
func All() []interface{} {
    return []interface{}{
        &User{},
        &Conversation{},
        &ConversationMember{},
        &Message{},
    }
}

