-- ============================================================
-- RINCO ScyllaDB Keyspaces
-- ============================================================
-- Production uses NetworkTopologyStrategy with replication 3
-- across datacenters. For developer-mode (single node) we use
-- SimpleStrategy with RF=1 so the cluster can bootstrap.
-- SCYLLA_CLUSTER_NAME must be set via env.
-- ============================================================

CREATE KEYSPACE IF NOT EXISTS chat
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 3
  }
  AND durable_writes = true;

CREATE KEYSPACE IF NOT EXISTS webrtc
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 3
  }
  AND durable_writes = true;

CREATE KEYSPACE IF NOT EXISTS leads
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 3
  }
  AND durable_writes = true;

CREATE KEYSPACE IF NOT EXISTS events
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 3
  }
  AND durable_writes = true;

CREATE KEYSPACE IF NOT EXISTS audit
  WITH replication = {
    'class': 'NetworkTopologyStrategy',
    'replication_factor': 3
  }
  AND durable_writes = true;
