//! Tests for chat-engine configuration and error types.

use chat_engine::config::Config;
use chat_engine::error::ChatError;
use std::time::Duration;

#[test]
fn config_env_or_uses_fallback() {
    std::env::remove_var("CHAT_HTTP_ADDR");
    let cfg = Config::from_env().expect("from_env should not fail with defaults");
    assert_eq!(cfg.http_addr, "0.0.0.0:8080");
}

#[test]
fn config_env_override_http_addr() {
    std::env::set_var("CHAT_HTTP_ADDR", "127.0.0.1:9999");
    let cfg = Config::from_env().expect("from_env ok");
    assert_eq!(cfg.http_addr, "127.0.0.1:9999");
    std::env::remove_var("CHAT_HTTP_ADDR");
}

#[test]
fn config_default_vals_present() {
    std::env::remove_var("CHAT_HTTP_ADDR");
    let cfg = Config::from_env().unwrap();
    assert!(cfg.http_addr.starts_with("0.0.0.0"));
    assert!(!cfg.scylla_url.is_empty());
    assert!(!cfg.valkey_url.is_empty());
    assert!(!cfg.nats_url.is_empty());
}

#[test]
fn config_signal_db_key_has_default() {
    let cfg = Config::from_env().unwrap();
    assert!(!cfg.signal_db_key.is_empty());
}

#[test]
fn config_bool_hardware_rng_default_true() {
    std::env::remove_var("CHAT_SIGNAL_USE_HARDWARE_RNG");
    let cfg = Config::from_env().unwrap();
    // Default is true per config.rs
    assert!(cfg.signal_use_hardware_rng);
}

#[test]
fn config_bool_hardware_rng_explicit_false() {
    std::env::set_var("CHAT_SIGNAL_USE_HARDWARE_RNG", "false");
    let cfg = Config::from_env().unwrap();
    assert!(!cfg.signal_use_hardware_rng);
    std::env::remove_var("CHAT_SIGNAL_USE_HARDWARE_RNG");
}

#[test]
fn config_shutdown_timeout_default() {
    std::env::remove_var("CHAT_SHUTDOWN_TIMEOUT_SECS");
    let cfg = Config::from_env().unwrap();
    assert_eq!(cfg.shutdown_timeout, Duration::from_secs(30));
}

#[test]
fn config_shutdown_timeout_custom() {
    std::env::set_var("CHAT_SHUTDOWN_TIMEOUT_SECS", "60");
    let cfg = Config::from_env().unwrap();
    assert_eq!(cfg.shutdown_timeout, Duration::from_secs(60));
    std::env::remove_var("CHAT_SHUTDOWN_TIMEOUT_SECS");
}

#[test]
fn config_shutdown_timeout_invalid_falls_back() {
    std::env::set_var("CHAT_SHUTDOWN_TIMEOUT_SECS", "not-a-number");
    let cfg = Config::from_env().unwrap();
    // Falls back to 30
    assert_eq!(cfg.shutdown_timeout, Duration::from_secs(30));
    std::env::remove_var("CHAT_SHUTDOWN_TIMEOUT_SECS");
}

// =============================================================================
// ChatError
// =============================================================================

#[test]
fn chat_error_display_config() {
    let e = ChatError::Config("bad".to_string());
    assert!(e.to_string().contains("configuration"));
    assert!(e.to_string().contains("bad"));
}

#[test]
fn chat_error_display_scylla() {
    let e = ChatError::Scylla("boom".to_string());
    assert!(e.to_string().contains("scylla"));
}

#[test]
fn chat_error_display_valkey() {
    let e = ChatError::Valkey("cache miss".to_string());
    assert!(e.to_string().contains("valkey"));
}

#[test]
fn chat_error_display_nats() {
    let e = ChatError::Nats("disconnected".to_string());
    assert!(e.to_string().contains("nats"));
}

#[test]
fn chat_error_display_websocket() {
    let e = ChatError::WebSocket("protocol".to_string());
    assert!(e.to_string().contains("websocket"));
}

#[test]
fn chat_error_display_rpc() {
    let e = ChatError::Rpc("rpc failed".to_string());
    assert!(e.to_string().contains("rpc"));
}

#[test]
fn chat_error_display_crypto() {
    let e = ChatError::Crypto("decrypt failed".to_string());
    assert!(e.to_string().contains("crypto"));
}

#[test]
fn chat_error_display_invalid_request() {
    let e = ChatError::InvalidRequest("bad input".to_string());
    assert!(e.to_string().contains("invalid request"));
}

#[test]
fn chat_error_display_unauthorized() {
    let e = ChatError::Unauthorized("token expired".to_string());
    assert!(e.to_string().contains("unauthorized"));
}

#[test]
fn chat_error_display_not_found() {
    let e = ChatError::NotFound("msg-123".to_string());
    assert!(e.to_string().contains("not found"));
    assert!(e.to_string().contains("msg-123"));
}

#[test]
fn chat_error_from_anyhow() {
    let any: anyhow::Error = anyhow::anyhow!("oh no");
    let ce: ChatError = any.into();
    assert!(ce.to_string().contains("oh no"));
}

#[test]
fn chat_error_from_serde_json() {
    let je: serde_json::Error = serde_json::from_str::<i32>("not a number").unwrap_err();
    let ce: ChatError = je.into();
    // Either InvalidRequest or original error string is shown
    assert!(!ce.to_string().is_empty());
}

#[test]
fn chat_error_debug_format() {
    let e = ChatError::Config("x".to_string());
    let dbg = format!("{:?}", e);
    assert!(dbg.contains("Config"));
}
