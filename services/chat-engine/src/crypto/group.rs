//! Group chat sender keys (a la Signal Group Protocol).
//!
//! Each member generates a chain key; on membership change a fresh chain is
//! re-distributed via pairwise Signal sessions established with `x3dh_*`.

use chacha20poly1305::{
    aead::{Aead, Payload},
    Key as ChaKey, XChaCha20Poly1305, XNonce,
};
use rand::RngCore;
use sha2::Sha256;
use uuid::Uuid;
use zeroize::Zeroize;
use hkdf::Hkdf;

use crate::error::{ChatError, ChatResult};

#[derive(Debug, Clone, Zeroize)]
#[zeroize(drop)]
pub struct ChainKey([u8; 32]);

impl ChainKey {
    pub fn random() -> Self {
        let mut b = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut b);
        Self(b)
    }

    pub fn from_bytes(b: [u8; 32]) -> Self {
        Self(b)
    }

    pub fn as_bytes(&self) -> &[u8; 32] {
        &self.0
    }

    /// Advance the chain by one step, returning the new chain key + a fresh
    /// message key for the i-th message.
    pub fn next(&self) -> (ChainKey, [u8; 32]) {
        let hk = Hkdf::<Sha256>::new(None, &self.0);
        let mut next = [0u8; 32];
        let mut mk = [u8::MAX; 32];
        hk.expand(b"GroupChain", &mut next).expect("32 < 255");
        hk.expand(b"GroupMsg", &mut mk).expect("32 < 255");
        (ChainKey(next), mk)
    }
}

/// Per-member sender state inside a group.
#[derive(Debug, Clone)]
pub struct SenderState {
    pub member: Uuid,
    pub chain: ChainKey,
}

impl SenderState {
    pub fn new(member: Uuid, chain: ChainKey) -> Self {
        Self { member, chain }
    }
}

/// In-memory representation of a group. The persistent form lives in ScyllaDB.
#[derive(Debug, Clone, Default)]
pub struct Group {
    pub group_id: Uuid,
    pub senders: Vec<SenderState>,
}

impl Group {
    /// Replace every member's chain key. Called when membership changes.
    pub fn redistribute(&mut self) {
        for s in &mut self.senders {
            s.chain = ChainKey::random();
        }
    }

    /// Encrypt a message using the sender's chain key.
    pub fn encrypt(
        &mut self,
        member: Uuid,
        plaintext: &[u8],
    ) -> ChatResult<GroupCiphertext> {
        let s = self
            .senders
            .iter_mut()
            .find(|s| s.member == member)
            .ok_or_else(|| ChatError::Crypto("unknown member".into()))?;
        let (next_chain, mk) = s.chain.next();
        s.chain = next_chain;
        let cipher = XChaCha20Poly1305::new(ChaKey::from_slice(&mk));
        let mut nonce = [0u8; 24];
        rand::thread_rng().fill_bytes(&mut nonce);
        let ct = cipher
            .encrypt(
                XNonce::from_slice(&nonce),
                chacha20poly1305::aead::Payload { msg: plaintext, aad: &[] },
            )
            .map_err(|e| ChatError::Crypto(e.to_string()))?;
        Ok(GroupCiphertext {
            member,
            nonce: nonce.to_vec(),
            ciphertext: ct,
        })
    }
}

#[derive(Debug, Clone)]
pub struct GroupCiphertext {
    pub member: Uuid,
    pub nonce: Vec<u8>,
    pub ciphertext: Vec<u8>,
}
