//! Multi-bucket S3 client (hot / cold / deep).
//!
//! Wraps the `rust-s3` crate to talk to MinIO and AWS S3. The buckets are
//! configured up-front and selected by `tier_for_object(...)`.

use bytes::Bytes;
use s3::bucket::Bucket;
use s3::creds::Credentials;
use s3::region::Region;
use std::sync::Arc;
use tokio::io::AsyncWriteExt;
use tracing::info;

use crate::config::Config;
use crate::error::{RecordingError, RecordingResult};

/// Where an object lives.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Tier {
    Hot,
    Cold,
    Deep,
}

impl Tier {
    pub fn from_age_days(days: i64, cfg: &Config) -> Self {
        if days >= cfg.deep_tier_days {
            Tier::Deep
        } else if days >= cfg.cold_tier_days {
            Tier::Cold
        } else {
            Tier::Hot
        }
    }
}

/// The storage client. Cheap to clone (Arc inside).
#[derive(Clone)]
pub struct TieredStorage {
    hot: Arc<Bucket>,
    cold: Arc<Bucket>,
    deep: Option<Arc<Bucket>>,
    config: Config,
}

impl TieredStorage {
    /// Build a new client from the application config.
    pub fn new(config: Config) -> RecordingResult<Self> {
        let region = Region::Custom {
            region: config.s3_region.clone(),
            endpoint: config.s3_endpoint.clone(),
        };
        let creds = Credentials::new(
            Some(&config.s3_access_key),
            Some(&config.s3_secret_key),
            None,
            None,
            None,
        )
        .map_err(|e| RecordingError::Storage(e.to_string()))?;
        let make_bucket = |name: &str| -> RecordingResult<Arc<Bucket>> {
            let mut b = Bucket::new(name, region.clone(), creds.clone())
                .map_err(|e| RecordingError::Storage(e.to_string()))?;
            // Use path-style so MinIO works out of the box.
            b = b.with_path_style();
            // `with_path_style` returns a `Box<Bucket>` – unwrap and Arc it.
            Ok(Arc::new(*b))
        };
        Ok(Self {
            hot: make_bucket(&config.s3_hot_bucket)?,
            cold: make_bucket(&config.s3_cold_bucket)?,
            deep: config
                .s3_deep_bucket
                .as_ref()
                .map(|n| make_bucket(n))
                .transpose()?,
            config,
        })
    }

    fn bucket(&self, tier: Tier) -> &Bucket {
        match tier {
            Tier::Hot => &self.hot,
            Tier::Cold => &self.cold,
            Tier::Deep => self.deep.as_deref().unwrap_or(&self.cold),
        }
    }

    /// Stream a chunk upload to the hot tier. Returns the object key.
    pub async fn upload_stream(
        &self,
        key: &str,
        mut stream: impl futures::Stream<Item = Bytes> + Unpin + Send,
    ) -> RecordingResult<String> {
        let tmp = tempfile::NamedTempFile::new()
            .map_err(|e| RecordingError::Storage(e.to_string()))?;
        let mut f = tmp.into_file();
        let mut total = 0u64;
        while let Some(chunk) = stream.next().await {
            f.write_all(&chunk)
                .await
                .map_err(|e| RecordingError::Storage(e.to_string()))?;
            total += chunk.len() as u64;
        }
        let _ = f.flush().await;
        let path = f.path().to_path_buf();
        let bytes = tokio::fs::read(&path)
            .await
            .map_err(|e| RecordingError::Storage(e.to_string()))?;
        self.bucket(Tier::Hot)
            .put_object(key, &bytes)
            .await
            .map_err(|e| RecordingError::Upload(e.to_string()))?;
        info!(key, bytes = total, "hot upload complete");
        Ok(format!("s3://{}/{}", self.config.s3_hot_bucket, key))
    }

    /// Generate a presigned download URL valid for `expires_secs`.
    pub async fn presign(&self, key: &str, tier: Tier, expires_secs: u32) -> RecordingResult<String> {
        let bucket = self.bucket(tier);
        let url = bucket
            .presign_get(key, expires_secs, None)
            .map_err(|e| RecordingError::Storage(e.to_string()))?;
        Ok(url)
    }

    /// Delete an object from the given tier.
    pub async fn delete(&self, key: &str, tier: Tier) -> RecordingResult<()> {
        self.bucket(tier)
            .delete_object(key)
            .await
            .map_err(|e| RecordingError::Storage(e.to_string()))?;
        Ok(())
    }

    /// Copy an object between tiers.
    pub async fn copy_to_tier(
        &self,
        key: &str,
        from: Tier,
        to: Tier,
    ) -> RecordingResult<()> {
        let from_bucket = self.bucket(from).name.to_string();
        let to_bucket = self.bucket(to);
        let copy_src = format!("{}/{}", from_bucket, key);
        // `copy_object` is a thin wrapper around the S3 PUT-object-copy API.
        // The exact signature depends on the version of `rust-s3`; we use
        // a generic `Result<StatusCode, _>` shape to keep the code compiling
        // against both 0.33 and 0.34.
        let resp = to_bucket
            .copy_object(&copy_src, key)
            .await
            .map_err(|e| RecordingError::Storage(e.to_string()))?;
        if resp.status_code() >= 300 {
            return Err(RecordingError::Storage(format!(
                "copy returned status {}",
                resp.status_code()
            )));
        }
        // Source is left in place; the metadata store flips the tier flag.
        Ok(())
    }
}
