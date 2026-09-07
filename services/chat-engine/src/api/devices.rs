//! Device / PreKey management.

use chrono::Utc;
use uuid::Uuid;

use crate::api::types::{Device, PreKeyBundle};
use crate::crypto::signal::IdentityKeyPair;
use crate::db::scylla::ScyllaStore;
use crate::error::ChatResult;

/// Register a brand-new device. Generates an initial batch of one-time prekeys
/// and stores the bundle server-side (private material encrypted with KEK).
pub async fn register_device(
    db: &ScyllaStore,
    user_id: Uuid,
    device_id: u64,
    identity: &IdentityKeyPair,
    prekey_count: usize,
) -> ChatResult<Device> {
    let signed = crate::crypto::signal::SignedPreKey::generate(identity, Utc::now().timestamp() as u64)?;
    let one_time = (0..prekey_count)
        .map(|_| crate::crypto::signal::OneTimePreKey::generate())
        .collect::<Result<Vec<_>, _>>()?;

    let device = Device {
        user_id,
        device_id,
        identity_pub: identity.public().to_bytes(),
        signed_prekey: signed.public_bytes(),
        signed_prekey_sig: signed.signature().to_vec(),
        one_time_prekeys: one_time.iter().map(|k| k.public_bytes()).collect(),
        registration_id: identity.registration_id(),
        last_seen: Utc::now(),
        name: None,
    };
    db.upsert_device(&device).await?;
    Ok(device)
}

/// Return a pre-key bundle that a peer can use to start a session.
pub async fn get_prekey_bundle(
    db: &ScyllaStore,
    user_id: Uuid,
    device_id: u64,
) -> ChatResult<PreKeyBundle> {
    let device = db
        .get_device(user_id, device_id)
        .await?
        .ok_or_else(|| crate::error::ChatError::NotFound("device not found".into()))?;
    // Consume one one-time prekey if available.
    let one_time = if let Some(first) = device.one_time_prekeys.first().cloned() {
        db.consume_one_time_prekey(user_id, device_id, &first).await?;
        Some(first)
    } else {
        None
    };
    Ok(PreKeyBundle {
        user_id,
        device_id,
        identity_pub: device.identity_pub,
        signed_prekey: device.signed_prekey,
        signed_prekey_sig: device.signed_prekey_sig,
        one_time_prekey: one_time,
    })
}

/// Upload a pre-key bundle for an existing device (used for replenishment).
pub async fn upload_prekey_bundle(
    db: &ScyllaStore,
    user_id: Uuid,
    device_id: u64,
    one_time_prekeys: Vec<Vec<u8>>,
) -> ChatResult<()> {
    db.append_one_time_prekeys(user_id, device_id, one_time_prekeys)
        .await
}
