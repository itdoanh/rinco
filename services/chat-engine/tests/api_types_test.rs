//! Tests for chat-engine API types (Channel, Message, etc.) - JSON roundtrip and constructors.

use chat_engine::api::types::{Channel, ChannelKind, Message, Reaction, Device, PreKey, Attachment};
use chrono::Utc;
use uuid::Uuid;

fn dummy_channel() -> Channel {
    Channel {
        id: Uuid::new_v4(),
        tenant_id: Uuid::nil(),
        name: Some("general".into()),
        topic: Some("announcements".into()),
        kind: ChannelKind::Public,
        members: vec![Uuid::new_v4(), Uuid::new_v4()],
        created_at: Utc::now(),
        updated_at: Utc::now(),
        archived: false,
    }
}

fn dummy_message(channel_id: Uuid, sender: Uuid) -> Message {
    Message {
        id: Uuid::new_v4(),
        tenant_id: Uuid::nil(),
        channel_id,
        sender_id: sender,
        sender_device_id: 1,
        recipient_device_id: None,
        ciphertext: b"super secret".to_vec(),
        ratchet_pub: vec![0u8; 32],
        msg_number: 1,
        attachments: vec![],
        reply_to: None,
        server_ts: Utc::now(),
        client_ts: None,
        edited: false,
        deleted: false,
    }
}

#[test]
fn channel_kind_serializes_snake_case() {
    let json = serde_json::to_string(&ChannelKind::Direct).unwrap();
    assert_eq!(json, "\"direct\"");

    let json = serde_json::to_string(&ChannelKind::Group).unwrap();
    assert_eq!(json, "\"group\"");

    let json = serde_json::to_string(&ChannelKind::Public).unwrap();
    assert_eq!(json, "\"public\"");

    let json = serde_json::to_string(&ChannelKind::Thread).unwrap();
    assert_eq!(json, "\"thread\"");
}

#[test]
fn channel_kind_deserializes_snake_case() {
    let k: ChannelKind = serde_json::from_str("\"direct\"").unwrap();
    assert_eq!(k, ChannelKind::Direct);

    let k: ChannelKind = serde_json::from_str("\"group\"").unwrap();
    assert_eq!(k, ChannelKind::Group);
}

#[test]
fn channel_serializes_to_json() {
    let ch = dummy_channel();
    let json = serde_json::to_string(&ch).unwrap();
    assert!(json.contains("\"kind\":\"public\""));
    assert!(json.contains("\"name\":\"general\""));
    assert!(!json.is_empty());
}

#[test]
fn channel_roundtrip_json() {
    let ch = dummy_channel();
    let json = serde_json::to_string(&ch).unwrap();
    let back: Channel = serde_json::from_str(&json).unwrap();
    assert_eq!(back.id, ch.id);
    assert_eq!(back.tenant_id, ch.tenant_id);
    assert_eq!(back.kind, ch.kind);
    assert_eq!(back.name, ch.name);
    assert_eq!(back.members.len(), ch.members.len());
}

#[test]
fn channel_with_no_members() {
    let mut ch = dummy_channel();
    ch.members = vec![];
    let json = serde_json::to_string(&ch).unwrap();
    let back: Channel = serde_json::from_str(&json).unwrap();
    assert_eq!(back.members.len(), 0);
}

#[test]
fn channel_archived_flag() {
    let mut ch = dummy_channel();
    ch.archived = true;
    let json = serde_json::to_string(&ch).unwrap();
    let back: Channel = serde_json::from_str(&json).unwrap();
    assert!(back.archived);
}

#[test]
fn message_serializes_with_ciphertext() {
    let ch = dummy_channel();
    let msg = dummy_message(ch.id, Uuid::new_v4());
    let json = serde_json::to_string(&msg).unwrap();
    // Server should never see plaintext
    assert!(!json.contains("plaintext"));
    // Ciphertext is encoded as a sequence
    assert!(json.contains("\"ciphertext\""));
}

#[test]
fn message_roundtrip() {
    let ch = dummy_channel();
    let msg = dummy_message(ch.id, Uuid::new_v4());
    let json = serde_json::to_string(&msg).unwrap();
    let back: Message = serde_json::from_str(&json).unwrap();
    assert_eq!(back.id, msg.id);
    assert_eq!(back.ciphertext, msg.ciphertext);
    assert_eq!(back.msg_number, msg.msg_number);
}

#[test]
fn message_with_reply() {
    let ch = dummy_channel();
    let mut msg = dummy_message(ch.id, Uuid::new_v4());
    let reply_id = Uuid::new_v4();
    msg.reply_to = Some(reply_id);
    let json = serde_json::to_string(&msg).unwrap();
    let back: Message = serde_json::from_str(&json).unwrap();
    assert_eq!(back.reply_to, Some(reply_id));
}

#[test]
fn message_edited_and_deleted_flags() {
    let ch = dummy_channel();
    let mut msg = dummy_message(ch.id, Uuid::new_v4());
    msg.edited = true;
    msg.deleted = true;
    let json = serde_json::to_string(&msg).unwrap();
    let back: Message = serde_json::from_str(&json).unwrap();
    assert!(back.edited);
    assert!(back.deleted);
}

#[test]
fn message_with_attachments() {
    let ch = dummy_channel();
    let mut msg = dummy_message(ch.id, Uuid::new_v4());
    msg.attachments.push(Attachment {
        id: "att-1".into(),
        filename: "test.pdf".into(),
        content_type: "application/pdf".into(),
        size_bytes: 1024,
        url: "https://example.com/test.pdf".into(),
        encryption_key: Some(vec![1, 2, 3]),
    });
    let json = serde_json::to_string(&msg).unwrap();
    let back: Message = serde_json::from_str(&json).unwrap();
    assert_eq!(back.attachments.len(), 1);
    assert_eq!(back.attachments[0].id, "att-1");
}

#[test]
fn reaction_emoji_default() {
    let r = Reaction {
        emoji: "👍".into(),
        user_id: Uuid::new_v4(),
        message_id: Uuid::new_v4(),
        created_at: Utc::now(),
    };
    let json = serde_json::to_string(&r).unwrap();
    let back: Reaction = serde_json::from_str(&json).unwrap();
    assert_eq!(back.emoji, "👍");
}

#[test]
fn device_serialize() {
    let d = Device {
        id: Uuid::new_v4(),
        user_id: Uuid::new_v4(),
        device_name: "iPhone 15".into(),
        platform: "ios".into(),
        push_token: Some("token-123".into()),
        last_seen: Some(Utc::now()),
        created_at: Utc::now(),
    };
    let json = serde_json::to_string(&d).unwrap();
    assert!(json.contains("\"device_name\":\"iPhone 15\""));
    assert!(json.contains("\"platform\":\"ios\""));
    let back: Device = serde_json::from_str(&json).unwrap();
    assert_eq!(back.device_name, "iPhone 15");
}

#[test]
fn device_no_push_token() {
    let d = Device {
        id: Uuid::new_v4(),
        user_id: Uuid::new_v4(),
        device_name: "Desktop".into(),
        platform: "web".into(),
        push_token: None,
        last_seen: None,
        created_at: Utc::now(),
    };
    let json = serde_json::to_string(&d).unwrap();
    let back: Device = serde_json::from_str(&json).unwrap();
    assert_eq!(back.push_token, None);
    assert_eq!(back.last_seen, None);
}

#[test]
fn prekey_serialize() {
    let pk = PreKey {
        id: 42,
        public_key: vec![0u8; 32],
        signature: Some(vec![1u8; 64]),
        created_at: Utc::now(),
    };
    let json = serde_json::to_string(&pk).unwrap();
    let back: PreKey = serde_json::from_str(&json).unwrap();
    assert_eq!(back.id, 42);
    assert_eq!(back.public_key.len(), 32);
}

#[test]
fn attachment_serialize() {
    let a = Attachment {
        id: "att-1".into(),
        filename: "test.pdf".into(),
        content_type: "application/pdf".into(),
        size_bytes: 1024,
        url: "https://example.com/test.pdf".into(),
        encryption_key: Some(vec![1, 2, 3]),
    };
    let json = serde_json::to_string(&a).unwrap();
    let back: Attachment = serde_json::from_str(&json).unwrap();
    assert_eq!(back.size_bytes, 1024);
}

#[test]
fn channel_topic_none() {
    let mut ch = dummy_channel();
    ch.topic = None;
    let json = serde_json::to_string(&ch).unwrap();
    let back: Channel = serde_json::from_str(&json).unwrap();
    assert_eq!(back.topic, None);
}

#[test]
fn channel_kind_equality() {
    assert_eq!(ChannelKind::Direct, ChannelKind::Direct);
    assert_ne!(ChannelKind::Direct, ChannelKind::Group);
    assert_ne!(ChannelKind::Public, ChannelKind::Thread);
}
