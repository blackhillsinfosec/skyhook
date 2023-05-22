// TODO make names db name configurable
export const NAMES_DB_NAME = "names";
// TODO make chunks store name configurable
export const CHUNKS_STORE_NAME = "chunks";
// TODO make staging_enabled name configurable
export const STAGING_ENABLED_NAME = "staging_enabled";

export const BYTE = 8;
export const KILOBYTE = 128*BYTE;
export const MEGABYTE = 1024*KILOBYTE;
export const GIGABYTE = 1024*MEGABYTE;
export const REC_CHUNK_SIZE = 1;
export const MAX_CHUNK_SIZE = 100;
export const MAX_CHUNK_MB = MAX_CHUNK_SIZE*MEGABYTE;

export const MAX_WORKERS=5;
export const MAX_STAGING_REQ=8;
