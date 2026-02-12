-- Task 2.4: rollback vec_embedding migration
DROP INDEX IF EXISTS idx_vec_workspace;
DROP TABLE IF EXISTS vec_embedding;
