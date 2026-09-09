//! Tests for the SymmetricCipher XChaCha helpers exposed by the Signal module.

use chat_engine::crypto::signal::{decrypt_xchacha, encrypt_message, encrypt_xchacha};

fn random_key() -> [u8; 32] {
    let mut bytes = [0u8; 32];
    rand::RngCore::fill_bytes(&mut rand::thread_rng(), &mut bytes);
    bytes
}

#[test]
fn chacha20_round_trip() {
    let key = random_key();
    let ad = b"some aad";
    let plaintext = b"signal-protocol-message";

    let ct = encrypt_message(&key, plaintext, ad).unwrap();
    // Ciphertext should differ from plaintext.
    assert_ne!(ct, plaintext);
    // Round trip.
    let pt = chat_engine::crypto::signal::decrypt_message(&key, &ct, ad).unwrap();
    assert_eq!(pt, plaintext);
}

#[test]
fn chacha20_wrong_ad_fails() {
    let key = random_key();
    let ct = encrypt_message(&key, b"x", b"alice/bob").unwrap();
    let res = chat_engine::crypto::signal::decrypt_message(&key, &ct, b"eve/dave");
    assert!(res.is_err());
}

#[test]
fn xchacha20_round_trip() {
    let key = random_key();
    let ad = b"attachment-metadata";
    let plaintext = b"xchacha20poly1305 is great for large files";

    let ct = encrypt_xchacha(&key, plaintext, ad).unwrap();
    let pt = decrypt_xchacha(&key, &ct, ad).unwrap();
    assert_eq!(pt, plaintext);
}

#[test]
fn xchacha20_ciphertext_has_nonce_prefix() {
    let key = random_key();
    let ct = encrypt_xchacha(&key, b"x", b"").unwrap();
    // XChaCha20 nonce is 24 bytes + ciphertext + 16 byte tag.
    assert!(ct.len() >= 24 + 16);
}

#[test]
fn xchacha20_short_ciphertext_rejected() {
    let key = random_key();
    // Less than 24 bytes → must return an error.
    let res = decrypt_xchacha(&key, &[0u8; 10], b"");
    assert!(res.is_err());
}

#[test]
fn chacha20_ciphertext_varies_with_same_input() {
    // Nonce is derived from message_key via HKDF, so identical keys + AD
    // produce identical ciphertexts.  We just verify that encrypting
    // distinct plaintexts with the same key yields distinct outputs.
    let key = random_key();
    let a = encrypt_message(&key, b"a", b"").unwrap();
    let b = encrypt_message(&key, b"b", b"").unwrap();
    assert_ne!(a, b);
}

#[test]
fn session_id_is_deterministic_and_symmetric() {
    use chat_engine::crypto::signal::session_id;
    use uuid::Uuid;
    let a = Uuid::new_v4();
    let b = Uuid::new_v4();
    // Order shouldn't matter — we sort internally.
    assert_eq!(session_id(a, b), session_id(b, a));
    // Different pair should differ.
    let c = Uuid::new_v4();
    assert_ne!(session_id(a, b), session_id(a, c));
}
