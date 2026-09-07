//! Recording Service – main binary.

use std::sync::Arc;
use std::time::Duration;

use actix_web::{
    get, post, web, App, HttpResponse, HttpServer, Responder,
};
use anyhow::Context;
use recording_service::{
    api::recording_service::{
        DeleteRecordingRequest, GetRecordingRequest, GetTranscriptRequest, ListRecordingsRequest,
        StartRecordingRequest, StopRecordingRequest, UpdateTierRequest,
    },
    config::Config,
    observability,
    storage::{metadata::InMemoryMetadata, tiered::TieredStorage},
};
use serde::Deserialize;
use tracing::info;
use uuid::Uuid;

#[derive(Clone)]
struct AppState {
    config: Config,
    storage: Arc<TieredStorage>,
    meta: Arc<InMemoryMetadata>,
}

#[actix_web::main]
async fn main() -> anyhow::Result<()> {
    let cfg = Config::from_env().context("load config")?;
    let _telemetry = observability::init("recording-service", &cfg.otlp_endpoint)?;
    info!(addr = %cfg.http_addr, "recording-service starting");

    let storage = Arc::new(TieredStorage::new(cfg.clone()).context("init s3 client")?);
    let meta = Arc::new(InMemoryMetadata::new());

    // Background jobs.
    {
        let storage = storage.clone();
        let meta = meta.clone();
        let cfg = cfg.clone();
        tokio::spawn(async move {
            let mut ticker = tokio::time::interval(Duration::from_secs(3600));
            loop {
                ticker.tick().await;
                if let Err(e) = recording_service::jobs::transition::run_once(
                    storage.clone(),
                    meta.clone(),
                    Arc::new(cfg.clone()),
                )
                .await
                {
                    tracing::error!(error = %e, "tier transition sweep failed");
                }
            }
        });
    }

    let state = AppState { config: cfg.clone(), storage, meta };
    let data = web::Data::new(state);
    let bind = cfg.http_addr.clone();
    HttpServer::new(move || App::new().app_data(data.clone()).configure(routes))
        .bind(&bind)
        .with_context(|| format!("bind {}", bind))?
        .run()
        .await?;
    Ok(())
}

fn routes(cfg: &mut web::ServiceConfig) {
    cfg.service(healthz)
        .service(readyz)
        .service(metrics)
        .service(start_recording)
        .service(stop_recording)
        .service(get_recording)
        .service(list_recordings)
        .service(delete_recording)
        .service(get_transcript)
        .service(update_tier);
}

#[get("/healthz")]
async fn healthz() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({"status": "ok"}))
}

#[get("/readyz")]
async fn readyz() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({"status": "ok"}))
}

#[get("/metrics")]
async fn metrics() -> impl Responder {
    HttpResponse::Ok()
        .content_type("text/plain; version=0.0.4")
        .body(observability::render_metrics())
}

#[post("/v1/recordings/start")]
async fn start_recording(
    state: web::Data<AppState>,
    body: web::Json<StartRecordingRequest>,
) -> impl Responder {
    let svc = recording_service::api::recording_service::DefaultRecordingService;
    let gpu = state.config.gpu_available;
    let resp = svc
        .start(
            body.into_inner(),
            state.storage.clone(),
            state.meta.clone(),
            gpu,
        )
        .await;
    match resp {
        Ok(r) => HttpResponse::Created().json(r),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

#[post("/v1/recordings/stop")]
async fn stop_recording(
    state: web::Data<AppState>,
    body: web::Json<StopRecordingRequest>,
) -> impl Responder {
    let svc = recording_service::api::recording_service::DefaultRecordingService;
    match svc.stop(body.into_inner()).await {
        Ok(_) => HttpResponse::NoContent().finish(),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

#[get("/v1/recordings/{id}")]
async fn get_recording(
    state: web::Data<AppState>,
    id: web::Path<Uuid>,
) -> impl Responder {
    let svc = recording_service::api::recording_service::DefaultRecordingService;
    let req = GetRecordingRequest { recording_id: id.into_inner() };
    match svc.get(req, state.storage.clone(), state.meta.clone()).await {
        Ok(r) => HttpResponse::Ok().json(r),
        Err(e) => HttpResponse::NotFound().json(serde_json::json!({"error": e.to_string()})),
    }
}

#[derive(Deserialize)]
struct ListQuery {
    room_id: Option<Uuid>,
    #[serde(default = "default_limit")]
    limit: i32,
}

fn default_limit() -> i32 {
    50
}

#[get("/v1/recordings")]
async fn list_recordings(
    state: web::Data<AppState>,
    q: web::Query<ListQuery>,
) -> impl Responder {
    let svc = recording_service::api::recording_service::DefaultRecordingService;
    let req = ListRecordingsRequest { room_id: q.room_id, limit: q.limit };
    match svc.list(req, state.meta.clone()).await {
        Ok(r) => HttpResponse::Ok().json(r),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

#[post("/v1/recordings/{id}/delete")]
async fn delete_recording(
    state: web::Data<AppState>,
    id: web::Path<Uuid>,
) -> impl Responder {
    let svc = recording_service::api::recording_service::DefaultRecordingService;
    let req = DeleteRecordingRequest { recording_id: id.into_inner() };
    match svc.delete(req, state.storage.clone(), state.meta.clone()).await {
        Ok(_) => HttpResponse::NoContent().finish(),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}

#[get("/v1/recordings/{id}/transcript")]
async fn get_transcript(
    state: web::Data<AppState>,
    id: web::Path<Uuid>,
) -> impl Responder {
    let svc = recording_service::api::recording_service::DefaultRecordingService;
    let req = GetTranscriptRequest { recording_id: id.into_inner() };
    match svc.get_transcript(req, state.meta.clone()).await {
        Ok(r) => HttpResponse::Ok().json(r),
        Err(e) => HttpResponse::NotFound().json(serde_json::json!({"error": e.to_string()})),
    }
}

#[post("/v1/recordings/{id}/tier")]
async fn update_tier(
    state: web::Data<AppState>,
    id: web::Path<Uuid>,
    body: web::Json<serde_json::Value>,
) -> impl Responder {
    let svc = recording_service::api::recording_service::DefaultRecordingService;
    let tier = match body.get("tier").and_then(|v| v.as_str()) {
        Some("hot") => recording_service::storage::tiered::Tier::Hot,
        Some("cold") => recording_service::storage::tiered::Tier::Cold,
        Some("deep") => recording_service::storage::tiered::Tier::Deep,
        _ => return HttpResponse::BadRequest().json(serde_json::json!({"error": "invalid tier"})),
    };
    let req = UpdateTierRequest { recording_id: id.into_inner(), tier };
    match svc.update_tier(req, state.meta.clone()).await {
        Ok(_) => HttpResponse::NoContent().finish(),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()})),
    }
}
