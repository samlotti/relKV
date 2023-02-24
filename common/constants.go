package common

const (
	BADGER_FLAG_ALIAS = 1

	HEADER_B64_KEY              = "b64"
	HEADER_SKIP_KEY             = "skip"
	HEADER_MAX_KEY              = "max"
	HEADER_EXPLAIN_KEY          = "explain"
	HEADER_PREFIX_KEY           = "prefix"
	HEADER_VALUES_KEY           = "values"
	HEADER_ALIAS_KEY            = "aliases"
	HEADER_ALIAS_SEPARATOR      = ";"
	HEADER_SEGMENT_KEY          = "segments"
	HEADER_SEGMENT_SEPARATOR    = ":"
	RESP_HEADER_KVDB_FUNCTION   = "func"
	RESP_HEADER_DUPLICATE_ERROR = "duplicate_key"
	RESP_HEADER_ERROR_MSG       = "error_msg"
)
