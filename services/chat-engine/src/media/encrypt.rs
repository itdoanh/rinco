//! Media attachment encryption.
//!
//! Large media files are encrypted with a fresh 32-byte DEK (XChaCha20-Poly1305).
//! The DEK is then wrapped with the Signal session's current message key so only
//! the recipient can decrypt it.

use rand::RngCore;
use uuid::Uuid;

use crate::api::types::Attachment;
use crate::crypto::signal;
use crate::db::scylla::ScyllaStore;
use crate::error::ChatResult;

/// Result of encrypting a media attachment.
pub struct EncryptedMedia {
    /// Server-side metadata that should be persisted alongside the message.
    pub attachment: Attachment,
    /// Raw ciphertext – upload to S3/MinIO as-is.
    pub ciphertext: Vec<u8>,
}

/// Encrypt a media payload.
///
/// `session_message_key` is the active message key of the Signal session at the
/// time of upload. Wrapping the DEK with the same key means the recipient can
/// decrypt the DEK using the same Double Ratchet step that delivered the
/// attachment pointer.
pub fn encrypt_media(
    msg_id: Uuid,
    s3_key: String,
    mime: String,
    plaintext: &[u8],
    session_message_key: &[u8; 32],
) -> ChatResult<EncryptedMedia> {
    let mut dek = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut dek);
    let ciphertext = signal::encrypt_xchacha(&dek, plaintext, msg_id.as_bytes())?;
    let encrypted_dek = signal::encrypt_message(session_message_key, &dek, msg_id.as_bytes())?;
    Ok(EncryptedMedia {
        attachment: Attachment {
            msg_id,
            s3_key,
            mime,
            size: plaintext.len() as u64,
            encrypted_dek,
            encrypted_size: ciphertext.len() as u64,
            thumbnail_url: None,
        },
        ciphertext,
    })
}

/// Decrypt a media payload using the recipient's session key.
pub fn decrypt_media(
    session_message_key: &[u8; 32],
    attachment: &Attachment,
    ciphertext: &[u8],
) -> ChatResult<Vec<u8>> {
    let dek = signal::decrypt_message(
        session_message_key,
        &attachment.encrypted_dek,
        attachment.msg_id.as_bytes(),
    )?;
    if dek.len() != 32 {
        return Err(crate::error::ChatError::Crypto("invalid dek".into()));
    }
    let mut key = [0u8; 32];
    key.copy_from_slice(&dek);
    signal::decrypt_xchacha(&key, ciphertext, attachment.msg_id.as_bytes())
}

/// Persist attachment metadata.
pub async fn persist_metadata(db: &ScyllaStore, attachment: &Attachment) -> ChatResult<()> {
    db.insert_attachment(
        attachment.msg_id,
        &attachment.s3_key,
        &attachment.mime,
        attachment.size,
        &attachment.encrypted_dek,
        attachment.encrypted_size,
        attachment.thumbnail_url.as_deref(),
    )
    .await
}
