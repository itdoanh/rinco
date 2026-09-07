//! Tests for the Signal Protocol implementation.

use chat_engine::crypto::signal::{
    decrypt_message, encrypt_message, x3dh_initiator, x3dh_responder, IdentityKeyPair,
    OneTimePreKey, RatchetState, SignedPreKey, PreKeyBundleRef,
};
use x25519_dalek::StaticSecret;

#[test]
fn x3dh_responder_matches_initiator() {
    let mut rng = rand::thread_rng();

    // Bob: identity + signed prekey + one one-time prekey.
    let bob_id = IdentityKeyPair::generate(&mut rng);
    let bob_sp = SignedPreKey::generate(&bob_id, 1).unwrap();
    let bob_otpk = OneTimePreKey::generate().unwrap();
    let bob_otpk_pub = bob_otpk.public_bytes();
    let bob_otpk_pub_arr: [u8; 32] = bob_otpk_pub.clone().try_into().unwrap();
    let bob_id_pub_arr: [u8; 32] = bob_id.public_bytes().try_into().unwrap();
    let bob_sp_pub: [u8; 32] = bob_sp.public_bytes().try_into().unwrap();

    // Alice: identity + ephemeral.
    let alice_id = IdentityKeyPair::generate(&mut rng);
    let mut eph_bytes = [0u8; 32];
    rand::RngCore::fill_bytes(&mut rng, &mut eph_bytes);
    let alice_eph = StaticSecret::from(eph_bytes);

    // X3DH as Alice.
    let bundle = PreKeyBundleRef {
        identity: bob_id_pub_arr,
        signed_prekey: bob_sp_pub,
        one_time_prekey: Some(bob_otpk_pub_arr),
        _lifetime: std::marker::PhantomData,
    };
    let alice_secrets = x3dh_initiator(&alice_id, &bundle, &alice_eph).unwrap();

    // X3DH as Bob.
    let mut eph_bytes2 = [0u8; 32];
    rand::RngCore::fill_bytes(&mut rng, &mut eph_bytes2);
    let alice_eph_pub: [u8; 32] = x25519_dalek::PublicKey::from(&alice_eph).to_bytes();
    let bob_secrets = x3dh_responder(
        &bob_id,
        &bob_sp,
        Some(&bob_otpk),
        bob_id_pub_arr,
        alice_eph_pub,
    )
    .unwrap();

    assert_eq!(alice_secrets.root_key, bob_secrets.root_key);
    assert_eq!(alice_secrets.chain_key, bob_secrets.chain_key);
}

#[test]
fn double_ratchet_round_trip() {
    let mut rng = rand::thread_rng();
    let alice = IdentityKeyPair::generate(&mut rng);
    let bob = IdentityKeyPair::generate(&mut rng);
    let bob_sp = SignedPreKey::generate(&bob, 1).unwrap();
    let alice_eph = StaticSecret::from(rand::random::<[u8; 32]>());
    let bundle = PreKeyBundleRef {
        identity: bob.public_bytes().try_into().unwrap(),
        signed_prekey: bob_sp.public_bytes().try_into().unwrap(),
        one_time_prekey: None,
        _lifetime: std::marker::PhantomData,
    };
    let init = x3dh_initiator(&alice, &bundle, &alice_eph).unwrap();
    let mut alice_ratchet = RatchetState::new(init);

    let mk = alice_ratchet.next_message_key();
    let plaintext = b"hello bob – alice over signal!";
    let ad = b"alice/bob/session1";
    let ct = encrypt_message(&mk, plaintext, ad).unwrap();
    let pt = decrypt_message(&mk, &ct, ad).unwrap();
    assert_eq!(pt, plaintext);
}

#[test]
fn constant_time_replay_protection() {
    use chat_engine::crypto::signal::monotonic_within_window;
    assert!(monotonic_within_window(100, 50, 100));
    assert!(!monotonic_within_window(40, 50, 100));
    assert!(!monotonic_within_window(200, 50, 100));
}
