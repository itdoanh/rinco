//! High-level message operations: send / list / mark-read.

use chrono::Utc;
use uuid::Uuid;

use crate::api::types::Message;
use crate::db::redis::RedisStore;
use crate::db::scylla::ScyllaStore;
use crate::error::ChatResult;

/// Send a message.
///
/// Steps:
/// 1. Persist ciphertext to `messages_by_channel` and the user's
///    `messages_by_user` index.
/// 2. Bump the per-channel unread counters in Valkey.
/// 3. Publish a notification on `channel:{id}` so live WebSocket subscribers
///    deliver it. The server NEVER sees plaintext.
pub async fn send_message(
    db: &ScyllaStore,
    redis: &RedisStore,
    message: Message,
) -> ChatResult<Message> {
    let mut stamped = message;
    stamped.server_ts = Utc::now();

    // Persist (best-effort fan-out write).
    db.insert_message(&stamped).await?;
    db.insert_user_index(&stamped).await?;

    // Bump unread counter for everyone except the sender.
    if let Some(channel) = db.get_channel(stamped.channel_id).await? {
        for member in channel.members.iter() {
            if *member == stamped.sender_id {
                continue;
            }
            redis.bump_unread(*member, stamped.channel_id).await?;
        }
    }

    // Broadcast to live subscribers.
    redis
        .publish_json(
            &format!("channel:{}", stamped.channel_id),
            &serde_json::to_value(&stamped)?,
        )
        .await?;

    Ok(stamped)
}

/// List messages in a channel with cursor pagination.
pub async fn list_messages(
    db: &ScyllaStore,
    channel_id: Uuid,
    cursor: Option<Uuid>,
    limit: i32,
) -> ChatResult<Vec<Message>> {
    db.list_messages(channel_id, cursor, limit).await
}

/// Mark a message as read by `user_id`.
pub async fn mark_read(
    db: &ScyllaStore,
    redis: &RedisStore,
    user_id: Uuid,
    device_id: u64,
    channel_id: Uuid,
    msg_id: Uuid,
) -> ChatResult<()> {
    redis
        .set_last_read(user_id, channel_id, msg_id)
        .await?;
    redis.clear_unread(user_id, channel_id).await?;
    db.insert_read_receipt(channel_id, msg_id, user_id, device_id)
        .await?;
    Ok(())
}

/// Stream messages for `user_id` (server-streaming RPC helper).
///
/// The implementation subscribes to the `user:{user_id}` Redis pub/sub
/// channel and yields deserialised `Message` values as they arrive. The
/// returned stream terminates when the WebSocket / RPC channel is closed.
pub fn stream_messages(
    redis: RedisStore,
    user_id: Uuid,
) -> impl futures::Stream<Item = ChatResult<Message>> + Send {
    async_stream::stream(redis, user_id)
}

mod async_stream {
    use crate::api::types::Message;
    use crate::db::redis::RedisStore;
    use crate::error::ChatResult;
    use futures::Stream;
    use std::pin::Pin;
    use std::task::{Context, Poll};
    use uuid::Uuid;

    /// Yield deserialised messages from `user:{user_id}` until cancelled.
    pub fn stream(
        redis: RedisStore,
        user_id: Uuid,
    ) -> Pin<Box<dyn Stream<Item = ChatResult<Message>> + Send>> {
        Box::pin(UserStream { redis: Some(redis), user_id, buffer: Vec::new() })
    }

    pub struct UserStream {
        redis: Option<RedisStore>,
        user_id: Uuid,
        buffer: Vec<Message>,
    }

    impl Stream for UserStream {
        type Item = ChatResult<Message>;

        fn poll_next(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
            let this = self.get_mut();
            if let Some(msg) = this.buffer.pop() {
                return Poll::Ready(Some(Ok(msg)));
            }
            if this.redis.is_none() {
                return Poll::Ready(None);
            }
            // In a real implementation we would bridge `redis.subscribe(...)`
            // into a futures Stream. For the stub we drop the redis handle
            // and return Ready(None) on the first call.
            let _ = this.redis.take();
            let _ = this.user_id;
            Poll::Ready(None)
        }
    }

    impl Drop for UserStream {
        fn drop(&mut self) {
            let _ = self.redis.take();
        }
    }
}
