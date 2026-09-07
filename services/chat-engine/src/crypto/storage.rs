//! At-rest encryption of Signal private material.
//!
//! A user-supplied passphrase is stretched with Argon2id to a KEK, which in
//! turn wraps a per-device DEK. The DEK wraps each private key with AES-256-GCM.

use aes_gcm::{
    aead::{Aead, KeyInit, Payload},
    Aes256Gcm, Nonce as GcmNonce,
};
use argon2::Argon2;
use rand::RngCore;
use zeroize::Zeroize;

use crate::error::{ChatError, ChatResult};

const NONCE_LEN: usize = 12;

#[derive(Debug, Clone, Zeroize)]
#[zeroize(drop)]
pub struct WrappedKey {
    pub nonce: [u8; NONCE_LEN],
    pub ciphertext: Vec<u8>,
}

/// Derive a 32-byte key from a passphrase using Argon2id.
pub fn derive_kek(passphrase: &str, salt: &[u8]) -> ChatResult<[u8; 32]> {
    let mut output = [0u8; 32];
    let argon = Argon2::default();
    argon
        .hash_password_into(passphrase.as_bytes(), salt, &mut output)
        .map_err(|e| ChatError::Crypto(e.to_string()))?;
    Ok(output)
}

/// Wrap a per-device DEK with the KEK derived from the passphrase.
pub fn wrap_dek(kek: &[u8; 32], dek: &[u8; 32]) -> ChatResult<WrappedKey> {
    let cipher = Aes256Gcm::new_from_slice(kek).map_err(|e| ChatError::Crypto(e.to_string()))?;
    let mut nonce = [0u8; NONCE_LEN];
    rand::thread_rng().fill_bytes(&mut nonce);
    let ct = cipher
        .encrypt(
            GcmNonce::from_slice(&nonce),
            Payload { msg: dek, aad: b"rinco-dek" },
        )
        .map_err(|e| ChatError::Crypto(e.to_string()))?;
    Ok(WrappedKey { nonce, ciphertext: ct })
}

/// Unwrap a per-device DEK.
pub fn unwrap_dek(kek: &[u8; 32], wrapped: &WrappedKey) -> ChatResult<[u8; 32]> {
    let cipher = Aes256Gcm::new_from_slice(kek).map_err(|e| ChatError::Crypto(e.to_string()))?;
    let pt = cipher
        .decrypt(
            GcmNonce::from_slice(&wrapped.nonce),
            Payload { msg: &wrapped.ciphertext, aad: b"rinco-dek" },
        )
        .map_err(|e| ChatError::Crypto(e.to_string()))?;
    if pt.len() != 32 {
        return Err(ChatError::Crypto("dek length mismatch".into()));
    }
    let mut out = [0u8; 32];
    out.copy_from_slice(&pt);
    Ok(out)
}

/// Encrypt arbitrary private key material with a per-device DEK.
pub fn seal(dek: &[u8; 32], plaintext: &[u8]) -> ChatResult<WrappedKey> {
    let cipher = Aes256Gcm::new_from_slice(dek).map_err(|e| ChatError::Crypto(e.to_string()))?;
    let mut nonce = [0u8; NONCE_LEN];
    rand::thread_rng().fill_bytes(&mut nonce);
    let ct = cipher
        .encrypt(
            GcmNonce::from_slice(&nonce),
            Payload { msg: plaintext, aad: b"rinco-identity" },
        )
        .map_err(|e| ChatError::Crypto(e.to_string()))?;
    Ok(WrappedKey { nonce, ciphertext: ct })
}

/// Decrypt arbitrary private key material.
pub fn unseal(dek: &[u8; 32], wrapped: &WrappedKey) -> ChatResult<Vec<u8>> {
    let cipher = Aes256Gcm::new_from_slice(dek).map_err(|e| ChatError::Crypto(e.to_string()))?;
    cipher
        .decrypt(
            GcmNonce::from_slice(&wrapped.nonce),
            Payload { msg: &wrapped.ciphertext, aad: b"rinco-identity" },
        )
        .map_err(|e| ChatError::Crypto(e.to_string()))
}
