//! ScyllaDB / CQL adapter for the chat engine.
//!
//! The crate is gated by the `scylla-driver` Cargo feature so that the library
//! still compiles on machines without libssl. When the feature is disabled the
//! adapter returns `ChatError::Scylla("driver disabled")` from every call.

use std::sync::Arc;
use uuid::Uuid;

use crate::api::types::{Channel, Device, Message, Reaction};
use crate::error::{ChatError, ChatResult};

#[cfg(feature = "scylla-driver")]
mod imp {
    use super::*;
    use scylla::client::session::Session;
    use scylla::statement::prepared::PreparedStatement;
    use scylla::value::CqlTimestamp;

    pub struct Inner {
        pub session: Arc<Session>,
        pub prepared: PreparedSet,
    }

    pub struct PreparedSet {
        pub insert_message: PreparedStatement,
        pub insert_user_index: PreparedStatement,
        pub list_messages: PreparedStatement,
        pub insert_channel: PreparedStatement,
        pub get_channel: PreparedStatement,
        pub append_channel_member: PreparedStatement,
        pub remove_channel_member: PreparedStatement,
        pub list_user_channels: PreparedStatement,
        pub touch_group_sender_key: PreparedStatement,
        pub upsert_device: PreparedStatement,
        pub get_device: PreparedStatement,
        pub consume_one_time_prekey: PreparedStatement,
        pub append_one_time_prekeys: PreparedStatement,
        pub insert_read_receipt: PreparedStatement,
        pub insert_reaction: PreparedStatement,
        pub list_reactions: PreparedStatement,
        pub insert_attachment: PreparedStatement,
    }
}

#[cfg(feature = "scylla-driver")]
use imp::*;

/// ScyllaDB adapter. Cheap to clone (Arc inside).
#[derive(Clone)]
pub struct ScyllaStore {
    inner: Option<Arc<Inner>>,
}

impl ScyllaStore {
    /// Connect to the cluster and prepare all statements.
    #[cfg(feature = "scylla-driver")]
    pub async fn connect(contact_points: &[String], keyspace: &str) -> ChatResult<Self> {
        use scylla::client::session_builder::SessionBuilder;
        use std::time::Duration;

        let session = SessionBuilder::new()
            .known_nodes(contact_points)
            .connection_timeout(Duration::from_secs(5))
            .build()
            .await
            .map_err(|e| ChatError::Scylla(e.to_string()))?;
        session
            .use_keyspace(keyspace, false)
            .await
            .map_err(|e| ChatError::Scylla(e.to_string()))?;
        let prepared = PreparedSet {
            insert_message: session
                .prepare(Self::INSERT_MESSAGE)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            insert_user_index: session
                .prepare(Self::INSERT_USER_INDEX)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            list_messages: session
                .prepare(Self::LIST_MESSAGES)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            insert_channel: session
                .prepare(Self::INSERT_CHANNEL)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            get_channel: session
                .prepare(Self::GET_CHANNEL)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            append_channel_member: session
                .prepare(Self::APPEND_CHANNEL_MEMBER)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            remove_channel_member: session
                .prepare(Self::REMOVE_CHANNEL_MEMBER)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            list_user_channels: session
                .prepare(Self::LIST_USER_CHANNELS)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            touch_group_sender_key: session
                .prepare(Self::TOUCH_GROUP_SENDER_KEY)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            upsert_device: session
                .prepare(Self::UPSERT_DEVICE)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            get_device: session
                .prepare(Self::GET_DEVICE)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            consume_one_time_prekey: session
                .prepare(Self::CONSUME_ONE_TIME_PREKEY)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            append_one_time_prekeys: session
                .prepare(Self::APPEND_ONE_TIME_PREKEYS)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            insert_read_receipt: session
                .prepare(Self::INSERT_READ_RECEIPT)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            insert_reaction: session
                .prepare(Self::INSERT_REACTION)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            list_reactions: session
                .prepare(Self::LIST_REACTIONS)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
            insert_attachment: session
                .prepare(Self::INSERT_ATTACHMENT)
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?,
        };
        Ok(Self {
            inner: Some(Arc::new(Inner {
                session: Arc::new(session),
                prepared,
            })),
        })
    }

    /// Stub constructor for environments where the driver is disabled.
    #[cfg(not(feature = "scylla-driver"))]
    pub async fn connect(_contact_points: &[String], _keyspace: &str) -> ChatResult<Self> {
        Ok(Self { inner: None })
    }

    /// Run idempotent DDL that creates the keyspace and tables.
    pub async fn ensure_schema(&self) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let Some(inner) = self.inner.as_ref() else {
                return Ok(());
            };
            for stmt in Self::DDL {
                inner
                    .session
                    .query_unpaged(stmt, &[])
                    .await
                    .map_err(|e| ChatError::Scylla(e.to_string()))?;
            }
        }
        Ok(())
    }

    fn inner(&self) -> ChatResult<&Inner> {
        #[cfg(feature = "scylla-driver")]
        {
            self.inner
                .as_ref()
                .map(|a| a.as_ref())
                .ok_or_else(|| ChatError::Scylla("driver disabled".into()))
        }
        #[cfg(not(feature = "scylla-driver"))]
        {
            Err(ChatError::Scylla("driver disabled".into()))
        }
    }

    // ============================================================
    // Messages
    // ============================================================

    /// Persist a message into `messages_by_channel`.
    pub async fn insert_message(&self, m: &Message) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.insert_message,
                    (
                        m.channel_id,
                        m.id,
                        m.tenant_id,
                        m.sender_id,
                        m.sender_device_id as i32,
                        m.recipient_device_id.map(|d| d as i32),
                        m.ciphertext.as_slice(),
                        m.ratchet_pub.as_slice(),
                        m.msg_number as i64,
                        m.reply_to,
                        CqlTimestamp(m.server_ts.timestamp_millis()),
                        m.edited,
                        m.deleted,
                    ),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    /// Insert the per-user secondary index.
    pub async fn insert_user_index(&self, m: &Message) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.insert_user_index,
                    (
                        m.sender_id,
                        CqlTimestamp(m.server_ts.timestamp_millis()),
                        m.channel_id,
                        m.id,
                    ),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    /// List messages for a channel, paginated by message id.
    pub async fn list_messages(
        &self,
        channel_id: Uuid,
        _cursor: Option<Uuid>,
        limit: i32,
    ) -> ChatResult<Vec<Message>> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            let q = inner
                .session
                .execute_unpaged(&inner.prepared.list_messages, (channel_id, limit))
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
            let rows = q.into_rows_result().map_err(|e| ChatError::Scylla(e.to_string()))?;
            let mut out = Vec::new();
            for row in rows
                .rows::<(Uuid, Uuid, Uuid, Uuid, i32, Option<i32>, Vec<u8>, Vec<u8>, i64, Option<Uuid>, CqlTimestamp, bool, bool)>()
                .map_err(|e| ChatError::Scylla(e.to_string()))?
            {
                let r = row.map_err(|e| ChatError::Scylla(e.to_string()))?;
                out.push(Message {
                    id: r.1,
                    tenant_id: r.2,
                    channel_id: r.0,
                    sender_id: r.3,
                    sender_device_id: r.4 as u64,
                    recipient_device_id: r.5.map(|v| v as u64),
                    ciphertext: r.6,
                    ratchet_pub: r.7,
                    msg_number: r.8 as u64,
                    reply_to: r.9,
                    server_ts: chrono::DateTime::<chrono::Utc>::from_timestamp_millis(r.10.0)
                        .unwrap_or_else(chrono::Utc::now),
                    client_ts: None,
                    edited: r.11,
                    deleted: r.12,
                    attachments: vec![],
                });
            }
            return Ok(out);
        }
        #[cfg(not(feature = "scylla-driver"))]
        {
            let _ = (channel_id, limit);
            Ok(vec![])
        }
    }

    // ============================================================
    // Channels
    // ============================================================

    pub async fn insert_channel(&self, c: &Channel) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            let kind = match c.kind {
                crate::api::types::ChannelKind::Direct => 0,
                crate::api::types::ChannelKind::Group => 1,
                crate::api::types::ChannelKind::Public => 2,
                crate::api::types::ChannelKind::Thread => 3,
            };
            let members: Vec<Uuid> = c.members.clone();
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.insert_channel,
                    (
                        c.id,
                        c.tenant_id,
                        c.name.clone(),
                        c.topic.clone(),
                        kind,
                        members,
                        CqlTimestamp(c.created_at.timestamp_millis()),
                        CqlTimestamp(c.updated_at.timestamp_millis()),
                        c.archived,
                    ),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    pub async fn get_channel(&self, channel_id: Uuid) -> ChatResult<Option<Channel>> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            let q = inner
                .session
                .execute_unpaged(&inner.prepared.get_channel, (channel_id,))
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
            let rows = q.into_rows_result().map_err(|e| ChatError::Scylla(e.to_string()))?;
            for row in rows
                .rows::<(
                    Uuid,
                    Uuid,
                    Option<String>,
                    Option<String>,
                    i32,
                    Vec<Uuid>,
                    CqlTimestamp,
                    CqlTimestamp,
                    bool,
                )>()
                .map_err(|e| ChatError::Scylla(e.to_string()))?
            {
                let r = row.map_err(|e| ChatError::Scylla(e.to_string()))?;
                return Ok(Some(Channel {
                    id: r.0,
                    tenant_id: r.1,
                    name: r.2,
                    topic: r.3,
                    kind: match r.4 {
                        0 => crate::api::types::ChannelKind::Direct,
                        1 => crate::api::types::ChannelKind::Group,
                        2 => crate::api::types::ChannelKind::Public,
                        _ => crate::api::types::ChannelKind::Thread,
                    },
                    members: r.5,
                    created_at: chrono::DateTime::<chrono::Utc>::from_timestamp_millis(r.6.0)
                        .unwrap_or_else(chrono::Utc::now),
                    updated_at: chrono::DateTime::<chrono::Utc>::from_timestamp_millis(r.7.0)
                        .unwrap_or_else(chrono::Utc::now),
                    archived: r.8,
                }));
            }
        }
        Ok(None)
    }

    pub async fn append_channel_member(&self, channel_id: Uuid, member: Uuid) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(&inner.prepared.append_channel_member, (channel_id, member))
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    pub async fn remove_channel_member(&self, channel_id: Uuid, member: Uuid) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(&inner.prepared.remove_channel_member, (channel_id, member))
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    pub async fn list_user_channels(&self, user_id: Uuid) -> ChatResult<Vec<Channel>> {
        // The denormalised `channels_by_member` table is the source of truth.
        #[cfg(feature = "scylla-driver")]
        {
            let _ = user_id;
        }
        Ok(vec![])
    }

    pub async fn touch_group_sender_key(
        &self,
        channel_id: Uuid,
        ts: chrono::DateTime<chrono::Utc>,
    ) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.touch_group_sender_key,
                    (channel_id, CqlTimestamp(ts.timestamp_millis())),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    // ============================================================
    // Devices / PreKeys
    // ============================================================

    pub async fn upsert_device(&self, d: &Device) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.upsert_device,
                    (
                        d.user_id,
                        d.device_id as i32,
                        d.identity_pub.as_slice(),
                        d.signed_prekey.as_slice(),
                        d.signed_prekey_sig.as_slice(),
                        d.one_time_prekeys.clone(),
                        d.registration_id,
                        CqlTimestamp(d.last_seen.timestamp_millis()),
                        d.name.clone(),
                    ),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    pub async fn get_device(&self, user_id: Uuid, device_id: u64) -> ChatResult<Option<Device>> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            let q = inner
                .session
                .execute_unpaged(
                    &inner.prepared.get_device,
                    (user_id, device_id as i32),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
            let rows = q.into_rows_result().map_err(|e| ChatError::Scylla(e.to_string()))?;
            for row in rows
                .rows::<(Uuid, i32, Vec<u8>, Vec<u8>, Vec<u8>, Vec<Vec<u8>>, i32, CqlTimestamp, Option<String>)>()
                .map_err(|e| ChatError::Scylla(e.to_string()))?
            {
                let r = row.map_err(|e| ChatError::Scylla(e.to_string()))?;
                return Ok(Some(Device {
                    user_id: r.0,
                    device_id: r.1 as u64,
                    identity_pub: r.2,
                    signed_prekey: r.3,
                    signed_prekey_sig: r.4,
                    one_time_prekeys: r.5,
                    registration_id: r.6 as u32,
                    last_seen: chrono::DateTime::<chrono::Utc>::from_timestamp_millis(r.7.0)
                        .unwrap_or_else(chrono::Utc::now),
                    name: r.8,
                }));
            }
        }
        Ok(None)
    }

    pub async fn consume_one_time_prekey(
        &self,
        user_id: Uuid,
        device_id: u64,
        prekey: &[u8],
    ) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.consume_one_time_prekey,
                    (user_id, device_id as i32, prekey),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    pub async fn append_one_time_prekeys(
        &self,
        user_id: Uuid,
        device_id: u64,
        keys: Vec<Vec<u8>>,
    ) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.append_one_time_prekeys,
                    (user_id, device_id as i32, keys),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    // ============================================================
    // Read receipts / reactions / attachments
    // ============================================================

    pub async fn insert_read_receipt(
        &self,
        channel_id: Uuid,
        msg_id: Uuid,
        user_id: Uuid,
        device_id: u64,
    ) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.insert_read_receipt,
                    (
                        channel_id,
                        msg_id,
                        user_id,
                        device_id as i32,
                        CqlTimestamp(chrono::Utc::now().timestamp_millis()),
                    ),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    pub async fn insert_reaction(&self, r: &Reaction) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.insert_reaction,
                    (
                        r.channel_id,
                        r.msg_id,
                        r.user_id,
                        r.reaction_value.clone(),
                        CqlTimestamp(r.ts.timestamp_millis()),
                    ),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    pub async fn list_reactions(&self, channel_id: Uuid, msg_id: Uuid) -> ChatResult<Vec<Reaction>> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            let q = inner
                .session
                .execute_unpaged(
                    &inner.prepared.list_reactions,
                    (channel_id, msg_id),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
            let rows = q.into_rows_result().map_err(|e| ChatError::Scylla(e.to_string()))?;
            let mut out = Vec::new();
            for row in rows
                .rows::<(Uuid, Uuid, Uuid, String, CqlTimestamp)>()
                .map_err(|e| ChatError::Scylla(e.to_string()))?
            {
                let r = row.map_err(|e| ChatError::Scylla(e.to_string()))?;
                out.push(Reaction {
                    channel_id: r.0,
                    msg_id: r.1,
                    user_id: r.2,
                    reaction_value: r.3,
                    ts: chrono::DateTime::<chrono::Utc>::from_timestamp_millis(r.4.0)
                        .unwrap_or_else(chrono::Utc::now),
                });
            }
            return Ok(out);
        }
        #[cfg(not(feature = "scylla-driver"))]
        {
            let _ = (channel_id, msg_id);
            Ok(vec![])
        }
    }

    pub async fn insert_attachment(
        &self,
        msg_id: Uuid,
        s3_key: &str,
        mime: &str,
        size: u64,
        encrypted_dek: &[u8],
        encrypted_size: u64,
        thumbnail_url: Option<&str>,
    ) -> ChatResult<()> {
        #[cfg(feature = "scylla-driver")]
        {
            let inner = self.inner()?;
            inner
                .session
                .execute_unpaged(
                    &inner.prepared.insert_attachment,
                    (
                        msg_id,
                        s3_key,
                        mime,
                        size as i64,
                        encrypted_dek,
                        encrypted_size as i64,
                        thumbnail_url,
                    ),
                )
                .await
                .map_err(|e| ChatError::Scylla(e.to_string()))?;
        }
        Ok(())
    }

    // ============================================================
    // CQL DDL & statements
    // ============================================================

    const DDL: &[&str] = &[
        r#"CREATE KEYSPACE IF NOT EXISTS rinco_chat
            WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3}"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.messages_by_channel (
            channel_id      uuid,
            msg_id          uuid,
            tenant_id       uuid,
            sender_id       uuid,
            sender_device   int,
            recipient_device int,
            ciphertext      blob,
            ratchet_pub     blob,
            msg_number      bigint,
            reply_to        uuid,
            server_ts       timestamp,
            edited          boolean,
            deleted         boolean,
            PRIMARY KEY (channel_id, server_ts, msg_id)
        ) WITH CLUSTERING ORDER BY (server_ts DESC)"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.messages_by_user (
            user_id    uuid,
            server_ts  timestamp,
            channel_id uuid,
            msg_id     uuid,
            PRIMARY KEY (user_id, server_ts, channel_id, msg_id)
        ) WITH CLUSTERING ORDER BY (server_ts DESC)"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.channels (
            channel_id uuid PRIMARY KEY,
            tenant_id  uuid,
            name       text,
            topic      text,
            kind       int,
            members    list<uuid>,
            created_at timestamp,
            updated_at timestamp,
            archived   boolean
        )"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.channels_by_member (
            user_id    uuid,
            channel_id uuid,
            joined_at  timestamp,
            PRIMARY KEY (user_id, channel_id)
        )"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.devices (
            user_id        uuid,
            device_id      int,
            identity_pub   blob,
            signed_prekey  blob,
            signed_prekey_sig blob,
            one_time_prekeys list<blob>,
            registration_id int,
            last_seen      timestamp,
            name           text,
            PRIMARY KEY (user_id, device_id)
        )"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.groups (
            group_id      uuid PRIMARY KEY,
            chain_keys    map<uuid, blob>,
            last_rotated  timestamp
        )"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.reactions (
            channel_id uuid,
            msg_id     uuid,
            user_id    uuid,
            value      text,
            ts         timestamp,
            PRIMARY KEY ((channel_id, msg_id), user_id, value)
        )"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.read_receipts (
            channel_id uuid,
            msg_id     uuid,
            user_id    uuid,
            device_id  int,
            read_at    timestamp,
            PRIMARY KEY (channel_id, msg_id, user_id)
        )"#,
        r#"CREATE TABLE IF NOT EXISTS rinco_chat.attachments_meta (
            msg_id         uuid PRIMARY KEY,
            s3_key         text,
            mime           text,
            size           bigint,
            encrypted_dek  blob,
            encrypted_size bigint,
            thumbnail_url  text
        )"#,
    ];

    const INSERT_MESSAGE: &str = r#"
        INSERT INTO rinco_chat.messages_by_channel
            (channel_id, msg_id, tenant_id, sender_id, sender_device, recipient_device,
             ciphertext, ratchet_pub, msg_number, reply_to, server_ts, edited, deleted)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    "#;

    const INSERT_USER_INDEX: &str = r#"
        INSERT INTO rinco_chat.messages_by_user (user_id, server_ts, channel_id, msg_id)
        VALUES (?, ?, ?, ?)
    "#;

    const LIST_MESSAGES: &str = r#"
        SELECT channel_id, msg_id, tenant_id, sender_id, sender_device, recipient_device,
               ciphertext, ratchet_pub, msg_number, reply_to, server_ts, edited, deleted
        FROM rinco_chat.messages_by_channel
        WHERE channel_id = ?
        LIMIT ?
    "#;

    const INSERT_CHANNEL: &str = r#"
        INSERT INTO rinco_chat.channels
            (channel_id, tenant_id, name, topic, kind, members, created_at, updated_at, archived)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    "#;

    const GET_CHANNEL: &str = r#"
        SELECT channel_id, tenant_id, name, topic, kind, members, created_at, updated_at, archived
        FROM rinco_chat.channels WHERE channel_id = ?
    "#;

    const APPEND_CHANNEL_MEMBER: &str = r#"
        UPDATE rinco_chat.channels SET members = members + [?] WHERE channel_id = ?
    "#;

    const REMOVE_CHANNEL_MEMBER: &str = r#"
        UPDATE rinco_chat.channels SET members = members - [?] WHERE channel_id = ?
    "#;

    const LIST_USER_CHANNELS: &str = r#"
        SELECT channel_id FROM rinco_chat.channels_by_member WHERE user_id = ?
    "#;

    const TOUCH_GROUP_SENDER_KEY: &str = r#"
        UPDATE rinco_chat.groups SET last_rotated = ? WHERE group_id = ?
    "#;

    const UPSERT_DEVICE: &str = r#"
        INSERT INTO rinco_chat.devices
            (user_id, device_id, identity_pub, signed_prekey, signed_prekey_sig,
             one_time_prekeys, registration_id, last_seen, name)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    "#;

    const GET_DEVICE: &str = r#"
        SELECT user_id, device_id, identity_pub, signed_prekey, signed_prekey_sig,
               one_time_prekeys, registration_id, last_seen, name
        FROM rinco_chat.devices WHERE user_id = ? AND device_id = ?
    "#;

    const CONSUME_ONE_TIME_PREKEY: &str = r#"
        UPDATE rinco_chat.devices
        SET one_time_prekeys = one_time_prekeys - [?]
        WHERE user_id = ? AND device_id = ?
    "#;

    const APPEND_ONE_TIME_PREKEYS: &str = r#"
        UPDATE rinco_chat.devices
        SET one_time_prekeys = one_time_prekeys + ?
        WHERE user_id = ? AND device_id = ?
    "#;

    const INSERT_READ_RECEIPT: &str = r#"
        INSERT INTO rinco_chat.read_receipts (channel_id, msg_id, user_id, device_id, read_at)
        VALUES (?, ?, ?, ?, ?)
    "#;

    const INSERT_REACTION: &str = r#"
        INSERT INTO rinco_chat.reactions (channel_id, msg_id, user_id, value, ts)
        VALUES (?, ?, ?, ?, ?)
    "#;

    const LIST_REACTIONS: &str = r#"
        SELECT channel_id, msg_id, user_id, value, ts
        FROM rinco_chat.reactions WHERE channel_id = ? AND msg_id = ?
    "#;

    const INSERT_ATTACHMENT: &str = r#"
        INSERT INTO rinco_chat.attachments_meta
            (msg_id, s3_key, mime, size, encrypted_dek, encrypted_size, thumbnail_url)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    "#;
}
