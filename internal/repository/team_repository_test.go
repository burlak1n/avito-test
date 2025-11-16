package repository

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/reviewer-service/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=reviewer_test sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test: failed to connect to test database: %v", err)
		return nil
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Skipping test: failed to ping test database: %v", err)
		return nil
	}

	// Создаем таблицы для тестов
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS teams (
			team_name VARCHAR(255) PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS users (
			user_id VARCHAR(255) PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			team_name VARCHAR(255) NOT NULL REFERENCES teams(team_name),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create test tables: %v", err)
		return nil
	}

	return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
	_, _ = db.Exec("DELETE FROM users")
	_, _ = db.Exec("DELETE FROM teams")
}

func TestTeamRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer cleanupTestDB(t, db)

	repo := NewTeamRepository(db)

	tests := []struct {
		name          string
		team          *models.Team
		expectedError error
		validate      func(t *testing.T, repo TeamRepository)
	}{
		{
			name: "successful creation with members",
			team: &models.Team{
				TeamName: "team-1",
				Members: []models.TeamMember{
					{UserID: "user-1", Username: "user1", IsActive: true},
					{UserID: "user-2", Username: "user2", IsActive: true},
				},
			},
			expectedError: nil,
			validate: func(t *testing.T, repo TeamRepository) {
				team, err := repo.GetByName("team-1")
				if err != nil {
					t.Errorf("expected team to be created, got error: %v", err)
					return
				}
				if team.TeamName != "team-1" {
					t.Errorf("expected team name 'team-1', got '%s'", team.TeamName)
				}
				if len(team.Members) != 2 {
					t.Errorf("expected 2 members, got %d", len(team.Members))
				}
			},
		},
		{
			name: "successful creation without members",
			team: &models.Team{
				TeamName: "team-empty",
				Members:  []models.TeamMember{},
			},
			expectedError: nil,
			validate: func(t *testing.T, repo TeamRepository) {
				team, err := repo.GetByName("team-empty")
				if err != nil {
					t.Errorf("expected team to be created, got error: %v", err)
					return
				}
				if len(team.Members) != 0 {
					t.Errorf("expected 0 members, got %d", len(team.Members))
				}
			},
		},
		{
			name: "duplicate team name",
			team: &models.Team{
				TeamName: "team-duplicate",
				Members:  []models.TeamMember{},
			},
			expectedError: nil,
			validate: func(t *testing.T, repo TeamRepository) {
				// Создаем команду второй раз
				duplicateTeam := &models.Team{
					TeamName: "team-duplicate",
					Members:  []models.TeamMember{},
				}
				err := repo.Create(duplicateTeam)
				if err == nil {
					t.Error("expected error when creating duplicate team")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestDB(t, db)
			err := repo.Create(tt.team)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, repo)
			}
		})
	}
}

func TestTeamRepository_GetByName(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	defer cleanupTestDB(t, db)

	repo := NewTeamRepository(db)

	tests := []struct {
		name          string
		setup         func(t *testing.T, repo TeamRepository)
		teamName      string
		expectedError error
		validate      func(t *testing.T, team *models.Team)
	}{
		{
			name: "successful get with members",
			setup: func(t *testing.T, repo TeamRepository) {
				team := &models.Team{
					TeamName: "team-with-members",
					Members: []models.TeamMember{
						{UserID: "user-1", Username: "user1", IsActive: true},
						{UserID: "user-2", Username: "user2", IsActive: false},
					},
				}
				if err := repo.Create(team); err != nil {
					t.Fatalf("failed to setup test data: %v", err)
				}
			},
			teamName:      "team-with-members",
			expectedError: nil,
			validate: func(t *testing.T, team *models.Team) {
				if team.TeamName != "team-with-members" {
					t.Errorf("expected team name 'team-with-members', got '%s'", team.TeamName)
				}
				if len(team.Members) != 2 {
					t.Errorf("expected 2 members, got %d", len(team.Members))
				}
				if team.Members[0].UserID != "user-1" {
					t.Errorf("expected first member user_id 'user-1', got '%s'", team.Members[0].UserID)
				}
			},
		},
		{
			name: "successful get without members",
			setup: func(t *testing.T, repo TeamRepository) {
				team := &models.Team{
					TeamName: "team-empty",
					Members:  []models.TeamMember{},
				}
				if err := repo.Create(team); err != nil {
					t.Fatalf("failed to setup test data: %v", err)
				}
			},
			teamName:      "team-empty",
			expectedError: nil,
			validate: func(t *testing.T, team *models.Team) {
				if len(team.Members) != 0 {
					t.Errorf("expected 0 members, got %d", len(team.Members))
				}
			},
		},
		{
			name:          "team not found",
			setup:         func(t *testing.T, repo TeamRepository) {},
			teamName:      "non-existent-team",
			expectedError: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestDB(t, db)
			if tt.setup != nil {
				tt.setup(t, repo)
			}

			team, err := repo.GetByName(tt.teamName)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, team)
			}
		})
	}
}
