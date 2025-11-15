package service

import "errors"

// Определяем константные ошибки для точного соответствия OpenAPI
var (
	ErrTeamExists        = errors.New("team already exists")
	ErrTeamNotFound      = errors.New("team not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrPRExists          = errors.New("PR already exists")
	ErrPRNotFound        = errors.New("PR not found")
	ErrAuthorNotFound    = errors.New("author not found")
	ErrPRMerged          = errors.New("cannot reassign on merged PR")
	ErrNotAssigned       = errors.New("reviewer is not assigned to this PR")
	ErrNoCandidate       = errors.New("no active replacement candidate in team")
	ErrInvalidTeamMember = errors.New("user is not a member of the specified team")
)
