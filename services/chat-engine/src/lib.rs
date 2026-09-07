//! Chat Engine library crate.
//!
//! Modules:
//! - `config`  – env-driven configuration
//! - `telemetry` – tracing/OTel + Prometheus initialization
//! - `api`     – domain types and high-level operations
//! - `db`      – ScyllaDB and Redis/Valkey adapters
//! - `crypto`  – Signal Protocol primitives + group sender keys
//! - `media`   – encrypted media attachment helpers
//! - `presence`– online + typing + read receipts
//! - `nats`    – NATS consumer for cross-service events
//! - `handlers`– HTTP / WebSocket / Connect-RPC handlers
//! - `error`   – unified error type

#![deny(rust_2018_idioms)]
#![warn(missing_docs)]
#![allow(clippy::result_large_err)]

pub mod api;
pub mod config;
pub mod crypto;
pub mod db;
pub mod error;
pub mod handlers;
pub mod media;
pub mod nats;
pub mod presence;
pub mod telemetry;

pub use config::Config;
pub use error::{ChatError, ChatResult};

use std::sync::Arc;
use uuid::Uuid;

use crate::crypto::signal::IdentityKeyPair;
use crate::db::redis::RedisStore;
use crate::db::scylla::ScyllaStore;

/// Shared application context injected into every handler.
pub struct AppContext {
    /// Owning tenant.
    pub tenant_id: Uuid,
    /// ScyllaDB adapter.
    pub db: ScyllaStore,
    /// Redis / Valkey adapter.
    pub redis: RedisStore,
    /// Per-device identity key cache. Production deployments source these from
    /// the encrypted key store (`crypto::storage`) and keep the unwrapped
    /// material in memory only for the lifetime of the process.
    identities: parking_lot::RwLock<std::collections::HashMap<(Uuid, u64), Arc<IdentityKeyPair>>>,
}

impl AppContext {
    /// Build a new context from a config + connected clients.
    pub fn new(tenant_id: Uuid, db: ScyllaStore, redis: RedisStore) -> Self {
        Self {
            tenant_id,
            db,
            redis,
            identities: parking_lot::RwLock::new(Default::default()),
        }
    }

    /// Lazily provision an identity key pair for the given `(user, device)`.
    /// Real implementations would pull the private material from
    /// `crypto::storage::unseal`.
    pub async fn identity_for(&self, user_id: &Uuid, device_id: u64) -> Option<Arc<IdentityKeyPair>> {
        {
            let read = self.identities.read();
            if let Some(k) = read.get(&(*user_id, device_id)) {
                return Some(k.clone());
            }
        }
        let mut rng = rand::thread_rng();
        let k = Arc::new(IdentityKeyPair::generate(&mut rng));
        self.identities
            .write()
            .insert((*user_id, device_id), k.clone());
        Some(k)
    }

    /// Liveness probe for ScyllaDB.
    pub async fn ping_scylla(&self) -> crate::error::ChatResult<()> {
        self.db.ensure_schema().await
    }

    /// Liveness probe for Valkey.
    pub async fn ping_redis(&self) -> crate::error::ChatResult<()> {
        let uuid = Uuid::nil();
        let _ = self.redis.get_presence(uuid).await?;
        Ok(())
    }
}
