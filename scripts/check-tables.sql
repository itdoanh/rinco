SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename;
SELECT trigger_name, event_object_table FROM information_schema.triggers WHERE trigger_schema='public' ORDER BY trigger_name;
