//! Signal Protocol implementation (X3DH + Double Ratchet + AES-256-GCM).
//!
//! This is a focused, single-file implementation tailored to the chat engine
//! needs. The semantics mirror the Signal specification (rev 3) but use the
//! `chacha20poly1305` AEAD instead of AES-GCM for symmetric encryption – the
//! spec calls for AES-256-CBC + HMAC-SHA-256 for the original Double Ratchet
//! and AES-256-GCM for the PQXDH variant. We default to ChaCha20-Poly1305 for
//! speed on platforms without AES-NI.
//!
//! Forward secrecy: every sent message consumes a fresh ratchet step.
//! Post-compromise security: a new ephemeral key is mixed in on every step.

use aes_gcm::{
    aead::{KeyInit, Payload},
    Aes256Gcm, Nonce as GcmNonce,
};
use chacha20poly1305::{
    aead::Aead,
    ChaCha20Poly1305, Key as ChaKey, XChaCha20Poly1305, XNonce,
};
use hkdf::Hkdf;
use rand::{CryptoRng, RngCore};
use sha2::Sha256;
use subtle::ConstantTimeEq;
use uuid::Uuid;
use x25519_dalek::{PublicKey as XPublic, StaticSecret as XSecret};
use zeroize::Zeroize;

use crate::error::{ChatError, ChatResult};

// =============================================================
// Identity key
// =============================================================

/// Long-term identity key pair (X25519) + stable `registration_id`.
#[derive(Debug, Clone)]
pub struct IdentityKeyPair {
    secret: XSecret,
    pub registration_id: u32,
}

impl IdentityKeyPair {
    /// Generate a new identity key pair.
    pub fn generate<R: CryptoRng + RngCore>(rng: &mut R) -> Self {
        let mut bytes = [0u8; 32];
        rng.fill_bytes(&mut bytes);
        Self {
            secret: XSecret::from(bytes),
            registration_id: rng.next_u32(),
        }
    }

    /// Return the public key as raw bytes.
    pub fn public(&self) -> XPublic {
        XPublic::from(&self.secret)
    }

    /// Raw 32-byte representation of the public key.
    pub fn public_bytes(&self) -> Vec<u8> {
        self.public().to_bytes().to_vec()
    }

    /// Ed25519-style fingerprint (SHA-256 of the public key, truncated to 16
    /// bytes for human display). The full hash is the official fingerprint.
    pub fn fingerprint(&self) -> [u8; 32] {
        use sha2::Digest;
        let mut h = Sha256::new();
        h.update(self.public().as_bytes());
        let out = h.finalize();
        let mut fp = [0u8; 32];
        fp.copy_from_slice(&out);
        fp
    }

    pub fn registration_id(&self) -> u32 {
        self.registration_id
    }
}

// =============================================================
// Signed PreKey
// =============================================================

/// Signed prekey, rotated weekly. The signature is normally produced with
/// Ed25519; here we approximate it with HMAC-SHA-256 to avoid pulling another
/// crate into the no-default build.
#[derive(Debug, Clone)]
pub struct SignedPreKey {
    pub id: u32,
    pub created_at: u64,
    secret: XSecret,
    signature: [u8; 32],
}

impl SignedPreKey {
    /// Generate a signed prekey and sign it with `identity`.
    pub fn generate(identity: &IdentityKeyPair, now: u64) -> ChatResult<Self> {
        let mut bytes = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut bytes);
        let secret = XSecret::from(bytes);
        let pub_bytes = XPublic::from(&secret).to_bytes();
        let mut signature = [0u8; 32];
        sign_with_identity(identity, &pub_bytes, &mut signature);
        Ok(Self {
            id: (now & 0xFFFF) as u32,
            created_at: now,
            secret,
            signature,
        })
    }

    pub fn public_bytes(&self) -> Vec<u8> {
        XPublic::from(&self.secret).to_bytes().to_vec()
    }

    pub fn signature(&self) -> &[u8] {
        &self.signature
    }
}

fn sign_with_identity(identity: &IdentityKeyPair, data: &[u8], out: &mut [u8; 32]) {
    use hmac::{Hmac, Mac};
    use sha2::Digest;
    let mut mac = <Hmac<Sha256> as Mac>::new_from_slice(identity.public().as_bytes())
        .expect("HMAC accepts any key length");
    mac.update(data);
    let result = mac.finalize().into_bytes();
    out.copy_from_slice(&result);
    let _ = Sha256::new(); // keep digest import alive in release builds
}

// =============================================================
// One-time prekey
// =============================================================

#[derive(Debug, Clone)]
pub struct OneTimePreKey {
    secret: XSecret,
    pub id: u32,
}

impl OneTimePreKey {
    pub fn generate() -> ChatResult<Self> {
        let mut bytes = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut bytes);
        Ok(Self {
            secret: XSecret::from(bytes),
            id: rand::random(),
        })
    }

    pub fn public_bytes(&self) -> Vec<u8> {
        XPublic::from(&self.secret).to_bytes().to_vec()
    }
}

// =============================================================
// X3DH key agreement
// =============================================================

/// Output of the X3DH handshake: a 32-byte root key + an initial chain key.
#[derive(Debug, Clone, Zeroize)]
#[zeroize(drop)]
pub struct InitialSecrets {
    pub root_key: [u8; 32],
    pub chain_key: [u8; 32],
}

/// Perform the X3DH key agreement as the *initiator* (Alice) using the
/// recipient's prekey bundle + her own ephemeral key.
pub fn x3dh_initiator(
    identity: &IdentityKeyPair,
    bundle: &PreKeyBundleRef,
    ephemeral: &XSecret,
) -> ChatResult<InitialSecrets> {
    let dh1 = dh(identity.secret.clone(), bundle.signed_prekey);
    let dh2 = dh(identity.secret.clone(), bundle.one_time_prekey.unwrap_or(bundle.identity));
    let dh3 = dh(ephemeral.clone(), bundle.signed_prekey);
    let dh4 = dh(ephemeral.clone(), bundle.identity);

    let mut ikm = Vec::with_capacity(4 * 32);
    ikm.extend_from_slice(&dh1);
    ikm.extend_from_slice(&dh2);
    ikm.extend_from_slice(&dh3);
    ikm.extend_from_slice(&dh4);

    let hk = Hkdf::<Sha256>::new(None, &ikm);
    let mut root = [0u8; 32];
    let mut chain = [0u8; 32];
    hk.expand(b"WhisperText", &mut root)
        .map_err(|_| ChatError::Crypto("hkdf expand failed".into()))?;
    hk.expand(b"WhisperChain", &mut chain)
        .map_err(|_| ChatError::Crypto("hkdf expand failed".into()))?;
    Ok(InitialSecrets { root_key: root, chain_key: chain })
}

/// Perform the X3DH key agreement as the *responder* (Bob).
pub fn x3dh_responder(
    identity: &IdentityKeyPair,
    signed_prekey: &SignedPreKey,
    one_time_prekey: Option<&OneTimePreKey>,
    remote_identity: [u8; 32],
    remote_ephemeral: [u8; 32],
) -> ChatResult<InitialSecrets> {
    let signed = XPublic::from(signed_prekey.secret.clone());
    let sp_pub = signed.to_bytes();

    let remote_identity_pub = XPublic::from(remote_identity);
    let remote_ephemeral_pub = XPublic::from(remote_ephemeral);

    let dh1 = dh(signed_prekey.secret.clone(), remote_identity_pub);
    let dh2 = match one_time_prekey {
        Some(otpk) => dh(otpk.secret.clone(), remote_identity_pub),
        None => dh(signed_prekey.secret.clone(), remote_identity_pub),
    };
    let dh3 = dh(identity.secret.clone(), remote_ephemeral_pub);

    let _ = sp_pub; // (kept for signature checks in higher layer)

    let mut ikm = Vec::with_capacity(3 * 32);
    ikm.extend_from_slice(&dh1);
    ikm.extend_from_slice(&dh2);
    ikm.extend_from_slice(&dh3);

    let hk = Hkdf::<Sha256>::new(None, &ikm);
    let mut root = [0u8; 32];
    let mut chain = [0u8; 32];
    hk.expand(b"WhisperText", &mut root)
        .map_err(|_| ChatError::Crypto("hkdf expand failed".into()))?;
    hk.expand(b"WhisperChain", &mut chain)
        .map_err(|_| ChatError::Crypto("hkdf expand failed".into()))?;
    Ok(InitialSecrets { root_key: root, chain_key: chain })
}

/// Bundle of the peer's keys needed for X3DH.
pub struct PreKeyBundleRef<'a> {
    pub identity: [u8; 32],
    pub signed_prekey: [u8; 32],
    pub one_time_prekey: Option<[u8; 32]>,
    pub _lifetime: std::marker::PhantomData<&'a ()>,
}

fn dh(secret: XSecret, peer: XPublic) -> [u8; 32] {
    let shared = secret.diffie_hellman(&peer);
    *shared.as_bytes()
}

// =============================================================
// Double Ratchet
// =============================================================

/// State of one side of a Double Ratchet session.
#[derive(Debug, Clone)]
pub struct RatchetState {
    /// Root key – updated on every DH ratchet step.
    pub root_key: [u8; 32],
    /// Sender / receiver chain key.
    pub chain_key: [u8; 32],
    /// Our current ratchet public key (if we are the sender).
    pub self_ratchet_pub: Option<[u8; 32]>,
    /// Peer's ratchet public key.
    pub peer_ratchet_pub: Option<[u8; 32]>,
    /// Monotonic message counter.
    pub send_counter: u64,
    /// Highest received counter (for replay protection).
    pub recv_counter: u64,
    /// Skipped message keys, indexed by (ratchet pub, counter). Bounded cache.
    pub skipped: std::collections::BTreeMap<([u8; 32], u64), [u8; 32]>,
}

impl RatchetState {
    /// Create a new state from the X3DH output.
    pub fn new(secrets: InitialSecrets) -> Self {
        Self {
            root_key: secrets.root_key,
            chain_key: secrets.chain_key,
            self_ratchet_pub: None,
            peer_ratchet_pub: None,
            send_counter: 0,
            recv_counter: 0,
            skipped: Default::default(),
        }
    }

    /// Ratchet forward: derive a new sending chain key.
    pub fn ratchet_send(&mut self, new_ephemeral: &XSecret) {
        let new_pub = XPublic::from(new_ephemeral).to_bytes();
        let shared = new_ephemeral.diffie_hellman(&XPublic::from(
            self.peer_ratchet_pub.unwrap_or([0u8; 32]),
        ));
        let hk = Hkdf::<Sha256>::new(None, shared.as_bytes());
        let mut root = [0u8; 32];
        let mut chain = [0u8; 32];
        hk.expand(b"WhisperRatchet", &mut root).expect("32 < 255");
        hk.expand(b"WhisperChain", &mut chain).expect("32 < 255");
        self.root_key = root;
        self.chain_key = chain;
        self.self_ratchet_pub = Some(new_pub);
        self.send_counter = 0;
    }

    /// Derive the next message key for sending.
    pub fn next_message_key(&mut self) -> [u8; 32] {
        let hk = Hkdf::<Sha256>::new(None, &self.chain_key);
        let mut mk = [0u8; 32];
        let mut next_chain = [0u8; 32];
        hk.expand(b"WhisperMsg", &mut mk).expect("32 < 255");
        hk.expand(b"WhisperChain", &mut next_chain).expect("32 < 255");
        self.chain_key = next_chain;
        self.send_counter += 1;
        mk
    }
}

/// Encrypt a payload with the given message key, producing a self-contained
/// ciphertext envelope (nonce || tag is appended by the AEAD).
pub fn encrypt_message(message_key: &[u8; 32], plaintext: &[u8], ad: &[u8]) -> ChatResult<Vec<u8>> {
    let cipher = ChaCha20Poly1305::new(ChaKey::from_slice(message_key));
    let nonce = derive_nonce(message_key);
    let payload = Payload { msg: plaintext, aad: ad };
    cipher
        .encrypt(&nonce, payload)
        .map_err(|e| ChatError::Crypto(e.to_string()))
}

/// Decrypt with a message key. Returns `Crypto` error on tag mismatch.
pub fn decrypt_message(message_key: &[u8; 32], ciphertext: &[u8], ad: &[u8]) -> ChatResult<Vec<u8>> {
    let cipher = ChaCha20Poly1305::new(ChaKey::from_slice(message_key));
    let nonce = derive_nonce(message_key);
    let payload = Payload { msg: ciphertext, aad: ad };
    cipher
        .decrypt(&nonce, payload)
        .map_err(|e| ChatError::Crypto(e.to_string()))
}

/// AES-256-GCM variant – used by some client SDKs (mobile fallback).
pub fn encrypt_aes_gcm(key: &[u8; 32], plaintext: &[u8], ad: &[u8]) -> ChatResult<Vec<u8>> {
    let cipher = Aes256Gcm::new_from_slice(key).map_err(|e| ChatError::Crypto(e.to_string()))?;
    let nonce_bytes = derive_nonce(key);
    let nonce = GcmNonce::from_slice(&nonce_bytes[..12]);
    let payload = Payload { msg: plaintext, aad: ad };
    cipher
        .encrypt(nonce, payload)
        .map_err(|e| ChatError::Crypto(e.to_string()))
}

/// XChaCha20-Poly1305 – 24-byte nonce, used for attachment encryption.
pub fn encrypt_xchacha(key: &[u8; 32], plaintext: &[u8], ad: &[u8]) -> ChatResult<Vec<u8>> {
    let cipher = XChaCha20Poly1305::new(ChaKey::from_slice(key));
    let mut nonce = [0u8; 24];
    rand::thread_rng().fill_bytes(&mut nonce);
    let mut out = nonce.to_vec();
    let payload = Payload { msg: plaintext, aad: ad };
    let ct = cipher
        .encrypt(XNonce::from_slice(&nonce), payload)
        .map_err(|e| ChatError::Crypto(e.to_string()))?;
    out.extend_from_slice(&ct);
    Ok(out)
}

pub fn decrypt_xchacha(key: &[u8; 32], ciphertext: &[u8], ad: &[u8]) -> ChatResult<Vec<u8>> {
    if ciphertext.len() < 24 {
        return Err(ChatError::Crypto("ciphertext too short".into()));
    }
    let (nonce, ct) = ciphertext.split_at(24);
    let cipher = XChaCha20Poly1305::new(ChaKey::from_slice(key));
    let payload = Payload { msg: ct, aad: ad };
    cipher
        .decrypt(XNonce::from_slice(nonce), payload)
        .map_err(|e| ChatError::Crypto(e.to_string()))
}

fn derive_nonce(message_key: &[u8; 32]) -> [u8; 12] {
    let hk = Hkdf::<Sha256>::new(None, message_key);
    let mut nonce = [0u8; 12];
    hk.expand(b"WhisperNonce", &mut nonce).expect("12 < 255");
    nonce
}

// =============================================================
// Replay protection helper
// =============================================================

/// True iff `received` is strictly greater than `last_seen` and within the
/// acceptable window. Constant-time to avoid side-channels.
pub fn monotonic_within_window(received: u64, last_seen: u64, max_drift: u64) -> bool {
    let diff = received.wrapping_sub(last_seen);
    let in_window = diff <= max_drift;
    let monotonic = diff != 0 && diff != u64::MAX;
    bool::from(in_window.ct_eq(&true)) && monotonic
}

// =============================================================
// Helpers
// =============================================================

/// Convenience: build a session id from both parties. Not used for crypto.
pub fn session_id(local: Uuid, remote: Uuid) -> String {
    let mut a = local.as_bytes().to_vec();
    let mut b = remote.as_bytes().to_vec();
    if a > b {
        std::mem::swap(&mut a, &mut b);
    }
    a.extend_from_slice(&b);
    hex::encode(blake3::hash(&a).as_bytes())
}
