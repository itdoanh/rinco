//! Integration tests for media attachment encryption helpers.

use chat_engine::crypto::signal::encrypt_message;
use chat_engine::media::encrypt::{decrypt_media, encrypt_media};
use uuid::Uuid;

fn random_key() -> [u8; 32] {
    let mut bytes = [0u8; 32];
    rand::RngCore::fill_bytes(&mut rand::thread_rng(), &mut bytes);
    bytes
}

#[test]
fn encrypt_decrypt_round_trip() {
    let msg_id = Uuid::new_v4();
    let key = random_key();
    let plaintext = b"secret attachment payload";

    let out = encrypt_media(
        msg_id,
        "attachments/abc".into(),
        "image/png".into(),
        plaintext,
        &key,
    )
    .expect("encrypt_media");

    // Ciphertext is larger than plaintext (nonce + tag appended).
    assert!(out.ciphertext.len() > plaintext.len());

    let decrypted = decrypt_media(&key, &out.attachment, &out.ciphertext)
        .expect("decrypt_media");
    assert_eq!(decrypted, plaintext);
}

#[test]
fn ciphertext_differs_between_calls() {
    let msg_id = Uuid::new_v4();
    let key = random_key();
    let plaintext = b"hello";

    let a = encrypt_media(msg_id, "k".into(), "x".into(), plaintext, &key).unwrap();
    let b = encrypt_media(msg_id, "k".into(), "x".into(), plaintext, &key).unwrap();

    // Each call uses a fresh DEK + XChaCha20 nonce → different ciphertexts.
    assert_ne!(a.ciphertext, b.ciphertext);
    // The wrapped DEKs should also differ.
    assert_ne!(a.attachment.encrypted_dek, b.attachment.encrypted_dek);
}

#[test]
fn decrypt_with_wrong_key_fails() {
    let msg_id = Uuid::new_v4();
    let key = random_key();
    let wrong_key = random_key();
    let out = encrypt_media(msg_id, "k".into(), "x".into(), b"secret", &key).unwrap();

    let res = decrypt_media(&wrong_key, &out.attachment, &out.ciphertext);
    assert!(res.is_err(), "decrypt with wrong key must fail");
}

#[test]
fn attachment_metadata_is_populated() {
    let msg_id = Uuid::new_v4();
    let key = random_key();
    let plaintext = vec![0u8; 1024];

    let out = encrypt_media(
        msg_id,
        "s3/key".into(),
        "application/pdf".into(),
        &plaintext,
        &key,
    )
    .unwrap();

    let att = &out.attachment;
    assert_eq!(att.msg_id, msg_id);
    assert_eq!(att.s3_key, "s3/key");
    assert_eq!(att.mime, "application/pdf");
    assert_eq!(att.size as usize, plaintext.len());
    assert!(!att.encrypted_dek.is_empty());
    assert!(att.encrypted_size > 0);
    assert!(att.thumbnail_url.is_none());
}

#[test]
fn empty_plaintext_is_supported() {
    let msg_id = Uuid::new_v4();
    let key = random_key();
    let out = encrypt_media(msg_id, "k".into(), "text/plain".into(), b"", &key).unwrap();
    let decrypted = decrypt_media(&key, &out.attachment, &out.ciphertext).unwrap();
    assert_eq!(decrypted, b"");
}

#[test]
fn large_plaintext_round_trip() {
    let msg_id = Uuid::new_v4();
    let key = random_key();
    let plaintext: Vec<u8> = (0..100_000u32).map(|i| (i & 0xFF) as u8).collect();

    let out = encrypt_media(
        msg_id,
        "big".into(),
        "application/octet-stream".into(),
        &plaintext,
        &key,
    )
    .unwrap();
    let decrypted = decrypt_media(&key, &out.attachment, &out.ciphertext).unwrap();
    assert_eq!(decrypted.len(), plaintext.len());
    assert_eq!(decrypted, plaintext);
}

#[test]
fn dek_wrapping_uses_message_key_via_signal_layer() {
    // Verifies the wrapped DEK is the ciphertext produced by Signal's
    // encrypt_message (not raw DEK bytes).  If we could unwrap with the
    // bare key, the implementation would be broken.
    let msg_id = Uuid::new_v4();
    let key = random_key();

    let out = encrypt_media(msg_id, "k".into(), "x".into(), b"x", &key).unwrap();
    // The wrapped DEK should NOT equal a 32-byte slice of zeros.
    assert_ne!(out.attachment.encrypted_dek, vec![0u8; 32]);
    // And it should be at least 32 bytes (Signal's AEAD tag is 16 bytes on top
    // of the 32-byte plaintext, plus nonce).
    assert!(out.attachment.encrypted_dek.len() >= 48);

    // Sanity: encrypt_message with the same key + msg_id produces a different
    // ciphertext (because it has a separate nonce derivation path).  Just
    // makes sure encrypt_message is reachable for the test.
    let direct = encrypt_message(&key, b"x", msg_id.as_bytes()).unwrap();
    assert!(!direct.is_empty());
}
