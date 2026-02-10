-- Migration 004: Rollback - Drop pipeline tables
DROP TABLE IF EXISTS pipeline_stage;
DROP TABLE IF EXISTS pipeline;
