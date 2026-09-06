//! Room - đại diện cho một meeting session.

use std::sync::Arc;
use tokio::sync::RwLock;
use chrono::Utc;

pub struct Room {
    pub id: String,
    pub tenant_id: String,
    pub host_user_id: String,
    pub title: String,
    pub max_participants: u32,
    pub created_at: i64,
    participants: Arc<RwLock<Vec<String>>>,
}

impl Room {
    pub fn new(
        id: String,
        tenant_id: String,
        host_user_id: String,
        title: String,
        max_participants: u32,
    ) -> Self {
        let mut initial_participants = Vec::new();
        initial_participants.push(host_user_id.clone());

        Self {
            id,
            tenant_id,
            host_user_id,
            title,
            max_participants,
            created_at: Utc::now().timestamp(),
            participants: Arc::new(RwLock::new(initial_participants)),
        }
    }

    pub async fn add_participant(&self, user_id: String) -> bool {
        let mut participants = self.participants.write().await;
        if participants.len() >= self.max_participants as usize {
            return false;
        }
        if !participants.contains(&user_id) {
            participants.push(user_id);
        }
        true
    }

    pub async fn remove_participant(&self, user_id: &str) {
        let mut participants = self.participants.write().await;
        participants.retain(|p| p != user_id);
    }

    pub async fn list_participants(&self) -> Vec<String> {
        self.participants.read().await.clone()
    }

    pub async fn participant_count(&self) -> usize {
        self.participants.read().await.len()
    }

    pub async fn is_empty(&self) -> bool {
        self.participants.read().await.is_empty()
    }
}
