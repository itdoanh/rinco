//! Daily tier transition worker.

use chrono::Utc;
use std::sync::Arc;
use tracing::info;

use crate::config::Config;
use crate::storage::metadata::InMemoryMetadata;
use crate::storage::tiered::{Tier, TieredStorage};

/// Scan metadata and move objects to the appropriate tier.
pub async fn run_once(
    storage: Arc<TieredStorage>,
    meta: Arc<InMemoryMetadata>,
    config: Arc<Config>,
) -> anyhow::Result<()> {
    let now = Utc::now();
    let rows = meta.list(None, 10_000);
    let mut hot_to_cold = 0u32;
    let mut cold_to_deep = 0u32;
    for row in rows {
        let age = (now - row.started_at).num_days();
        let target = Tier::from_age_days(age, &config);
        if target == row.tier {
            continue;
        }
        let from = row.tier;
        match storage.copy_to_tier(&row.s3_key, from, target).await {
            Ok(()) => {
                info!(key = %row.s3_key, from = ?from, to = ?target, "tier transition");
                if (from, target) == (Tier::Hot, Tier::Cold) {
                    hot_to_cold += 1;
                } else if (from, target) == (Tier::Cold, Tier::Deep) {
                    cold_to_deep += 1;
                }
                meta.update(row.id, |r| {
                    r.tier = target;
                    r.s3_bucket = match target {
                        Tier::Hot => config.s3_hot_bucket.clone(),
                        Tier::Cold => config.s3_cold_bucket.clone(),
                        Tier::Deep => config.s3_deep_bucket.clone().unwrap_or_else(|| config.s3_cold_bucket.clone()),
                    };
                });
            }
            Err(e) => {
                tracing::error!(error = %e, key = %row.s3_key, "tier transition failed");
            }
        }
    }
    info!(hot_to_cold, cold_to_deep, "tier transition sweep done");
    Ok(())
}
