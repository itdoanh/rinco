//! ScyllaDB client cho chat engine.

use anyhow::{Context, Result};
use scylla::client::session::Session;
use scylla::statement::prepared::PreparedStatement;
use std::sync::Arc;
use tokio::sync::OnceCell;

pub struct ScyllaClient {
    session: Arc<Session>,
    insert_message: OnceCell<PreparedStatement>,
    get_messages: OnceCell<PreparedStatement>,
}

impl ScyllaClient {
    pub async fn new(host: &str) -> Result<Self> {
        let session = scylla::client::session_builder::SessionBuilder::new()
            .known_node(format!("{}:9042", host))
            .use_keyspace("chat", false)
            .build()
            .await
            .context("connect to scylla")?;

        Ok(Self {
            session: Arc::new(session),
            insert_message: OnceCell::new(),
            get_messages: OnceCell::new(),
        })
    }

    pub async fn insert_message(
        &self,
        tenant_id: Uuid,
        conversation_id: Uuid,
        message_id: Uuid,
        sender_id: Uuid,
        content: &str,
        message_type: &str,
    ) -> Result<()> {
        let stmt = self
            .insert_message
            .get_or_try_init(|| async {
                self.session
                    .prepare(
                        "INSERT INTO chat.messages_by_conversation \
                         (tenant_id, conversation_id, message_id, sender_id, content, message_type) \
                         VALUES (?, ?, ?, ?, ?, ?)",
                    )
                    .await
                    .context("prepare insert")
            })
            .await?;

        self.session
            .execute_unpaged(
                stmt,
                (tenant_id, conversation_id, message_id, sender_id, content, message_type),
            )
            .await
            .context("execute insert")?;

        Ok(())
    }

    pub async fn get_messages(
        &self,
        tenant_id: Uuid,
        conversation_id: Uuid,
        limit: i32,
    ) -> Result<Vec<MessageRow>> {
        let stmt = self
            .get_messages
            .get_or_try_init(|| async {
                self.session
                    .prepare(
                        "SELECT tenant_id, conversation_id, message_id, sender_id, content, message_type \
                         FROM chat.messages_by_conversation \
                         WHERE tenant_id = ? AND conversation_id = ? \
                         LIMIT ?",
                    )
                    .await
                    .context("prepare get")
            })
            .await?;

        let query_result = self
            .session
            .execute_unpaged(stmt, (tenant_id, conversation_id, limit))
            .await
            .context("execute get")?;

        let rows = query_result.into_rows_result().context("rows result")?;
        let mut messages = Vec::new();

        for row in rows.rows::<scylla::value::scylla_unset_row>()? {
            let row = row?;
            let tenant_id: Uuid = row.columns[0]
                .as_uuid()
                .context("col 0 uuid")?;
            let conversation_id: Uuid = row.columns[1]
                .as_uuid()
                .context("col 1 uuid")?;
            let message_id: Uuid = row.columns[2]
                .as_uuid()
                .context("col 2 uuid")?;
            let sender_id: Uuid = row.columns[3]
                .as_uuid()
                .context("col 3 uuid")?;
            let content: String = row.columns[4]
                .as_text()
                .context("col 4 text")?
                .to_string();
            let message_type: String = row.columns[5]
                .as_text()
                .context("col 5 text")?
                .to_string();

            messages.push(MessageRow {
                tenant_id,
                conversation_id,
                message_id,
                sender_id,
                content,
                message_type,
            });
        }

        Ok(messages)
    }
}

#[derive(Debug, Clone)]
pub struct MessageRow {
    pub tenant_id: Uuid,
    pub conversation_id: Uuid,
    pub message_id: Uuid,
    pub sender_id: Uuid,
    pub content: String,
    pub message_type: String,
}
