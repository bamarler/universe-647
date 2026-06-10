-- reverse: create "task_tags" table
DROP TABLE "task_tags";
-- reverse: create index "task_status" to table: "tasks"
DROP INDEX "task_status";
-- reverse: create index "task_due_at" to table: "tasks"
DROP INDEX "task_due_at";
-- reverse: create index "task_defer_at" to table: "tasks"
DROP INDEX "task_defer_at";
-- reverse: create "tasks" table
DROP TABLE "tasks";
-- reverse: create "note_tags" table
DROP TABLE "note_tags";
-- reverse: create index "tag_name_kind" to table: "tags"
DROP INDEX "tag_name_kind";
-- reverse: create index "tag_kind" to table: "tags"
DROP INDEX "tag_kind";
-- reverse: create "tags" table
DROP TABLE "tags";
-- reverse: create index "saved_filters_name_key" to table: "saved_filters"
DROP INDEX "saved_filters_name_key";
-- reverse: create "saved_filters" table
DROP TABLE "saved_filters";
-- reverse: create "notes" table
DROP TABLE "notes";
-- reverse: create index "chunk_source_type_source_id_chunk_index" to table: "chunks"
DROP INDEX "chunk_source_type_source_id_chunk_index";
-- reverse: create index "chunk_source_type_item_due_at" to table: "chunks"
DROP INDEX "chunk_source_type_item_due_at";
-- reverse: create index "chunk_chunk_hash" to table: "chunks"
DROP INDEX "chunk_chunk_hash";
-- reverse: create "chunks" table
DROP TABLE "chunks";
