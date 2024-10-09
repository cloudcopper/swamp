package errors

// The Application return code errors
const (
	RetLayerFilesystemError          = 9
	RetLoadConfigError               = 10
	RetCreateDatabaseError           = 11
	RetMigrateDatabaseError          = 12
	RetCreateRepoRepositoryError     = 13
	RetCreateArtifactRepositoryError = 14
	RetCreateArtifactStorageError    = 15
	RetCreateChecksumServiceError    = 16
	RetCreateInputWatcherError       = 17
	RetCreateRepoRecordError         = 20
	RetCreateWebServerError          = 40
)
