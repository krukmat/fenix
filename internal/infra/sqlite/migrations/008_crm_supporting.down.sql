-- Migration 008: Rollback - Drop supporting tables
DROP TABLE IF EXISTS timeline_event;
DROP TABLE IF EXISTS attachment;
DROP TABLE IF EXISTS note;
DROP TABLE IF EXISTS activity;
