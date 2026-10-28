package gitstatus

import "errors"

var (
	errGitCommandTimeout    = errors.New("git command timed out")
	errGitStatusTimeout     = errors.New("git status timed out")
	errGitStatusOutputLimit = errors.New("git status output limit exceeded")
	errGitRenameRecord      = errors.New("incomplete git rename record")
	errGitFileLimit         = errors.New("git status file limit exceeded")
	errGitStatusRecord      = errors.New("invalid git status record")
	errGitStatusRecordKind  = errors.New("unknown git status record")
	errGitStatusColumns     = errors.New("invalid git status columns")
	errGitHeadUnavailable   = errors.New("git HEAD could not be read")
	errGitBlobSize          = errors.New("invalid git blob size")
	errGitBlobLimit         = errors.New("git blob exceeds read limit")
	errGitTreeRecord        = errors.New("invalid git tree record")
	errGitMetadata          = errors.New("unsupported git metadata")
	errGitConfigUnsupported = errors.New("unsupported git config")
	errGitConfigUnsafe      = errors.New("unsafe git config")
	errGitAlternates        = errors.New("unsupported git alternates")
	errGitSnapshotLimit     = errors.New("git metadata snapshot limit exceeded")
	errGitMetadataRace      = errors.New("git metadata changed during snapshot")
	errGitMetadataEntry     = errors.New("unsupported git metadata entry")
	errGitObjectFormat      = errors.New("unsupported git object format")
)
