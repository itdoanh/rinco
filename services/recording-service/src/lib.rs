//! Recording service library.

pub mod api;
pub mod config;
pub mod egress;
pub mod error;
pub mod jobs;
pub mod observability;
pub mod storage;

pub use config::Config;
pub use error::{RecordingError, RecordingResult};
