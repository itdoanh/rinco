//! Tests for the at-rest crypto storage helpers (KEK derivation + DEK wrapping).

use chat_engine::crypto::storage::{derive_kek, seal, unwrap_dek, unseal, wrap_dek};

fn random_kek() -> [u8; 32] {
    let mut bytes = [0u8; 32];
    rand::RngCore::fill_bytes(&mut rand::thread_rng(), &mut bytes);
    bytes
}

fn random_salt() -> Vec<u8> {
    let mut bytes = vec![0u8; 16];
    rand::RngCore::fill_bytes(&mut rand::thread_rng(), &mut bytes);
    bytes
}

#[test]
fn derive_kek_deterministic_with_same_salt() {
    let salt = random_salt();
    let k1 = derive_kek("correct horse battery staple", &salt).unwrap();
    let k2 = derive_kek("correct horse battery staple", &salt).unwrap();
    assert_eq!(k1, k2);
}

#[test]
fn derive_kek_changes_with_salt() {
    let k1 = derive_kek("same-passphrase", &random_salt()).unwrap();
    let k2 = derive_kek("same-passphrase", &random_salt()).unwrap();
    assert_ne!(k1, k2);
}

#[test]
fn derive_kek_changes_with_passphrase() {
    let salt = random_salt();
    let k1 = derive_kek("alpha", &salt).unwrap();
    let k2 = derive_kek("beta", &salt).unwrap();
    assert_ne!(k1, k2);
}

#[test]
fn wrap_unwrap_dek_round_trip() {
    let kek = random_kek();
    let dek = random_kek();
    let wrapped = wrap_dek(&kek, &dek).unwrap();
    let unwrapped = unwrap_dek(&kek, &wrapped).unwrap();
    assert_eq!(unwrapped, dek);
}

#[test]
fn wrap_dek_produces_different_ciphertext_each_call() {
    let kek = random_kek();
    let dek = random_kek();
    let a = wrap_dek(&kek, &dek).unwrap();
    let b = wrap_dek(&kek, &dek).unwrap();
    // Fresh nonce per call.
    assert_ne!(a.nonce, b.nonce);
    assert_ne!(a.ciphertext, b.ciphertext);
}

#[test]
fn unwrap_dek_with_wrong_kek_fails() {
    let kek = random_kek();
    let wrong = random_kek();
    let dek = random_kek();
    let wrapped = wrap_dek(&kek, &dek).unwrap();
    let res = unwrap_dek(&wrong, &wrapped);
    assert!(res.is_err(), "unwrap with wrong kek must fail");
}

#[test]
fn seal_unseal_round_trip() {
    let dek = random_kek();
    let plaintext = b"private identity key material (32 bytes)".to_vec();
    let wrapped = seal(&dek, &plaintext).unwrap();
    let out = unseal(&dek, &wrapped).unwrap();
    assert_eq!(out, plaintext);
}

#[test]
fn seal_unseal_with_wrong_dek_fails() {
    let dek = random_kek();
    let wrong = random_kek();
    let wrapped = seal(&dek, b"secret").unwrap();
    let res = unseal(&wrong, &wrapped);
    assert!(res.is_err(), "unseal with wrong dek must fail");
}

#[test]
fn seal_handles_empty_plaintext() {
    let dek = random_kek();
    let wrapped = seal(&dek, b"").unwrap();
    let out = unseal(&dek, &wrapped).unwrap();
    assert_eq!(out, b"");
}

#[test]
fn seal_handles_large_plaintext() {
    let dek = random_kek();
    let plaintext: Vec<u8> = (0..32 * 1024).map(|i| (i & 0xFF) as u8).collect();
    let wrapped = seal(&dek, &plaintext).unwrap();
    let out = unseal(&dek, &wrapped).unwrap();
    assert_eq!(out, plaintext);
}

#[test]
fn nonce_is_twelve_bytes() {
    let kek = random_kek();
    let dek = random_kek();
    let wrapped = wrap_dek(&kek, &dek).unwrap();
    assert_eq!(wrapped.nonce.len(), 12);
}
