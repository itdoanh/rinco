-- =====================================================================
-- STT service seed — sample transcription records.  Mirrors the data
-- produced by the STT service when meeting recordings are processed
-- end-to-end.
-- =====================================================================

CREATE TABLE IF NOT EXISTS rinco.ai.stt_transcripts (
    id           UUID PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    meeting_id   TEXT NOT NULL,
    language     TEXT NOT NULL,
    duration_s   REAL NOT NULL,
    sample_rate  INT NOT NULL DEFAULT 16000,
    model        TEXT NOT NULL DEFAULT 'large-v3',
    text         TEXT NOT NULL,
    segments     JSONB NOT NULL DEFAULT '[]'::jsonb,
    confidence   REAL NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO rinco.ai.stt_transcripts (id, tenant_id, meeting_id, language, duration_s, text, segments, confidence)
VALUES
    ('66666666-6666-6666-6666-000000000001',
     'demo-tenant', 'mtg-001', 'vi', 42.5,
     'Xin chào anh, em là Minh từ RINCO. Em muốn tư vấn cho anh gói CRM doanh nghiệp vừa và nhỏ. Hiện tại anh đang dùng phần mềm nào để quản lý khách hàng ạ?',
     '[{"start":0.0,"end":6.0,"text":"Xin chào anh, em là Minh từ RINCO.","speaker":"SPEAKER_00"},{"start":6.5,"end":14.0,"text":"Em muốn tư vấn gói CRM doanh nghiệp vừa và nhỏ.","speaker":"SPEAKER_00"},{"start":14.5,"end":28.0,"text":"Hiện tại anh đang dùng phần mềm nào để quản lý khách hàng ạ?","speaker":"SPEAKER_00"}]'::jsonb,
     0.93),

    ('66666666-6666-6666-6666-000000000002',
     'demo-tenant', 'mtg-002', 'en', 58.0,
     'Yesterday I shipped the lead-scoring batch endpoint. Today I''m working on the SHAP explanations. No blockers.',
     '[{"start":0.0,"end":9.0,"text":"Yesterday I shipped the lead-scoring batch endpoint.","speaker":"SPEAKER_01"},{"start":9.5,"end":18.0,"text":"Today I''m working on the SHAP explanations.","speaker":"SPEAKER_01"},{"start":18.5,"end":21.0,"text":"No blockers.","speaker":"SPEAKER_01"}]'::jsonb,
     0.95),

    ('66666666-6666-6666-6666-000000000003',
     'demo-tenant', 'mtg-003', 'vi', 75.0,
     'Bên em đã nhận được yêu cầu hỗ trợ. Anh vui lòng cho em biết mã lỗi hiển thị trên màn hình ạ. Em sẽ hướng dẫn anh khắc phục trong vài phút.',
     '[{"start":0.0,"end":10.0,"text":"Bên em đã nhận được yêu cầu hỗ trợ.","speaker":"SPEAKER_02"},{"start":10.5,"end":26.0,"text":"Anh vui lòng cho em biết mã lỗi hiển thị trên màn hình ạ.","speaker":"SPEAKER_02"},{"start":26.5,"end":40.0,"text":"Em sẽ hướng dẫn anh khắc phục trong vài phút.","speaker":"SPEAKER_02"}]'::jsonb,
     0.91)
ON CONFLICT (id) DO NOTHING;
