package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/reviewer-service/internal/handlers"
	"github.com/reviewer-service/internal/middleware"
	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/repository"
	"github.com/reviewer-service/internal/service"
)

var testDB *sql.DB

func setupTestDB(t *testing.T) *sql.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "reviewers")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("Database not reachable: %v", err)
	}

	return db
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func cleanupDB(t *testing.T, db *sql.DB) {
	queries := []string{
		"DELETE FROM pr_reviewers",
		"DELETE FROM pull_requests",
		"DELETE FROM users",
		"DELETE FROM teams",
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("Failed to cleanup: %v", err)
		}
	}
}

func setupTestServer(db *sql.DB) *httptest.Server {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	teamRepo := repository.NewTeamRepository(db)
	userRepo := repository.NewUserRepository(db)
	prRepo := repository.NewPullRequestRepository(db)
	statsRepo := repository.NewStatisticsRepository(db)

	teamService := service.NewTeamService(teamRepo, userRepo, prRepo, db, logger)
	userService := service.NewUserService(userRepo, prRepo, logger)
	prService := service.NewPullRequestService(prRepo, userRepo, logger)
	statsService := service.NewStatisticsService(statsRepo, logger)

	teamHandler := handlers.NewTeamHandler(teamService, logger)
	userHandler := handlers.NewUserHandler(userService, logger)
	prHandler := handlers.NewPullRequestHandler(prService, logger)
	statsHandler := handlers.NewStatisticsHandler(statsService, logger)

	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware(logger))

	r.HandleFunc("/team/add", teamHandler.AddTeam).Methods("POST")
	r.HandleFunc("/team/get", teamHandler.GetTeam).Methods("GET")
	r.HandleFunc("/team/deactivateMembers", teamHandler.DeactivateTeamMembers).Methods("POST")
	r.HandleFunc("/users/setIsActive", userHandler.SetUserActive).Methods("POST")
	r.HandleFunc("/users/getReview", userHandler.GetUserReviews).Methods("GET")
	r.HandleFunc("/pullRequest/create", prHandler.CreatePR).Methods("POST")
	r.HandleFunc("/pullRequest/merge", prHandler.MergePR).Methods("POST")
	r.HandleFunc("/pullRequest/reassign", prHandler.ReassignReviewer).Methods("POST")
	r.HandleFunc("/statistics", statsHandler.GetStatistics).Methods("GET")

	return httptest.NewServer(r)
}

func TestE2E_FullPRFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	cleanupDB(t, db)

	srv := setupTestServer(db)
	defer srv.Close()

	// Шаг 1: Создание команды
	teamPayload := map[string]interface{}{
		"team_name": "backend",
		"members": []map[string]interface{}{
			{"user_id": "user-1", "username": "Alice", "is_active": true},
			{"user_id": "user-2", "username": "Bob", "is_active": true},
			{"user_id": "user-3", "username": "Charlie", "is_active": true},
		},
	}

	resp := makeRequest(t, srv.URL+"/team/add", "POST", teamPayload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	var respWrapper struct {
		Team models.Team `json:"team"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respWrapper); err != nil {
		t.Fatalf("Failed to decode team response: %v, body: %s", err, readBody(t, resp))
	}
	if respWrapper.Team.TeamName != "backend" || len(respWrapper.Team.Members) != 3 {
		t.Fatalf("Invalid team response: %+v", respWrapper.Team)
	}

	// Шаг 2: Создание PR
	prPayload := map[string]string{
		"pull_request_id":   "pr-1",
		"pull_request_name": "Add feature",
		"author_id":         "user-1",
	}

	resp = makeRequest(t, srv.URL+"/pullRequest/create", "POST", prPayload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	var prRespWrapper struct {
		PR models.PullRequest `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prRespWrapper); err != nil {
		t.Fatalf("Failed to decode PR response: %v, body: %s", err, readBody(t, resp))
	}
	prResp := prRespWrapper.PR
	if prResp.Status != "OPEN" {
		t.Fatalf("Expected OPEN status, got %s", prResp.Status)
	}
	if len(prResp.AssignedReviewers) == 0 || len(prResp.AssignedReviewers) > 2 {
		t.Fatalf("Expected 1-2 reviewers, got %d", len(prResp.AssignedReviewers))
	}
	if contains(prResp.AssignedReviewers, "user-1") {
		t.Fatal("Author should not be assigned as reviewer")
	}

	// Шаг 3: Получение PR пользователя
	reviewer := prResp.AssignedReviewers[0]
	resp = makeRequest(t, srv.URL+"/users/getReview?user_id="+reviewer, "GET", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var reviewResp struct {
		PullRequests []models.PullRequestShort `json:"pull_requests"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reviewResp); err != nil {
		t.Fatalf("Failed to decode review response: %v, body: %s", err, readBody(t, resp))
	}
	if len(reviewResp.PullRequests) != 1 {
		t.Fatalf("Expected 1 PR, got %d", len(reviewResp.PullRequests))
	}

	// Шаг 4: Merge PR
	mergePayload := map[string]string{"pull_request_id": "pr-1"}
	resp = makeRequest(t, srv.URL+"/pullRequest/merge", "POST", mergePayload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	if err := json.NewDecoder(resp.Body).Decode(&prRespWrapper); err != nil {
		t.Fatalf("Failed to decode merge PR response: %v, body: %s", err, readBody(t, resp))
	}
	prResp = prRespWrapper.PR
	if prResp.Status != "MERGED" {
		t.Fatalf("Expected MERGED status, got %s", prResp.Status)
	}

	// Шаг 5: Попытка переназначения после merge (должна вернуть ошибку)
	reassignPayload := map[string]string{
		"pull_request_id": "pr-1",
		"old_user_id":     reviewer,
	}
	resp = makeRequest(t, srv.URL+"/pullRequest/reassign", "POST", reassignPayload)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409, got %d", resp.StatusCode)
	}

	// Шаг 6: Проверка статистики
	resp = makeRequest(t, srv.URL+"/statistics", "GET", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var stats models.Statistics
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode statistics response: %v, body: %s", err, readBody(t, resp))
	}
	if stats.Teams.Total != 1 {
		t.Errorf("Expected 1 team, got %d", stats.Teams.Total)
	}
	if stats.Users.Total != 3 {
		t.Errorf("Expected 3 users, got %d", stats.Users.Total)
	}
	if stats.PullRequests.Total != 1 {
		t.Errorf("Expected 1 PR, got %d", stats.PullRequests.Total)
	}
}

func TestE2E_ReassignReviewer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	cleanupDB(t, db)

	srv := setupTestServer(db)
	defer srv.Close()

	// Создание команды
	teamPayload := map[string]interface{}{
		"team_name": "frontend",
		"members": []map[string]interface{}{
			{"user_id": "dev-1", "username": "Dev1", "is_active": true},
			{"user_id": "dev-2", "username": "Dev2", "is_active": true},
			{"user_id": "dev-3", "username": "Dev3", "is_active": true},
			{"user_id": "dev-4", "username": "Dev4", "is_active": true},
		},
	}
	makeRequest(t, srv.URL+"/team/add", "POST", teamPayload)

	// Создание PR
	prPayload := map[string]string{
		"pull_request_id":   "pr-reassign",
		"pull_request_name": "Feature X",
		"author_id":         "dev-1",
	}
	resp := makeRequest(t, srv.URL+"/pullRequest/create", "POST", prPayload)
	var prRespWrapper struct {
		PR models.PullRequest `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prRespWrapper); err != nil {
		t.Fatalf("Failed to decode PR response: %v, body: %s", err, readBody(t, resp))
	}
	prResp := prRespWrapper.PR

	if len(prResp.AssignedReviewers) == 0 {
		t.Fatal("Expected at least 1 reviewer")
	}

	oldReviewer := prResp.AssignedReviewers[0]

	// Переназначение ревьювера
	reassignPayload := map[string]string{
		"pull_request_id": "pr-reassign",
		"old_user_id":     oldReviewer,
	}
	resp = makeRequest(t, srv.URL+"/pullRequest/reassign", "POST", reassignPayload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	var reassignResp struct {
		PR         models.PullRequest `json:"pr"`
		ReplacedBy string             `json:"replaced_by"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reassignResp); err != nil {
		t.Fatalf("Failed to decode reassign response: %v, body: %s", err, readBody(t, resp))
	}

	if reassignResp.ReplacedBy == oldReviewer {
		t.Error("New reviewer should be different from old reviewer")
	}
	if contains(reassignResp.PR.AssignedReviewers, oldReviewer) {
		t.Error("Old reviewer should be removed")
	}
	if !contains(reassignResp.PR.AssignedReviewers, reassignResp.ReplacedBy) {
		t.Error("New reviewer should be assigned")
	}
}

func TestE2E_InactiveUserNotAssigned(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	cleanupDB(t, db)

	srv := setupTestServer(db)
	defer srv.Close()

	// Создание команды с 1 активным и 1 неактивным
	teamPayload := map[string]interface{}{
		"team_name": "team-inactive",
		"members": []map[string]interface{}{
			{"user_id": "active-1", "username": "Active", "is_active": true},
			{"user_id": "inactive-1", "username": "Inactive", "is_active": false},
		},
	}
	makeRequest(t, srv.URL+"/team/add", "POST", teamPayload)

	// Создание PR от активного пользователя
	prPayload := map[string]string{
		"pull_request_id":   "pr-inactive",
		"pull_request_name": "Test",
		"author_id":         "active-1",
	}
	resp := makeRequest(t, srv.URL+"/pullRequest/create", "POST", prPayload)
	var prRespWrapper struct {
		PR models.PullRequest `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prRespWrapper); err != nil {
		t.Fatalf("Failed to decode PR response: %v, body: %s", err, readBody(t, resp))
	}
	prResp := prRespWrapper.PR

	// Не должно быть ревьюверов (автор - единственный активный)
	if len(prResp.AssignedReviewers) != 0 {
		t.Errorf("Expected 0 reviewers, got %d (inactive should not be assigned)", len(prResp.AssignedReviewers))
	}
}

func TestE2E_DeactivateTeamMembers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	cleanupDB(t, db)

	srv := setupTestServer(db)
	defer srv.Close()

	// Создание команды
	teamPayload := map[string]interface{}{
		"team_name": "deactivate-team",
		"members": []map[string]interface{}{
			{"user_id": "lead-1", "username": "Lead", "is_active": true},
			{"user_id": "member-1", "username": "Member1", "is_active": true},
			{"user_id": "member-2", "username": "Member2", "is_active": true},
			{"user_id": "member-3", "username": "Member3", "is_active": true},
		},
	}
	makeRequest(t, srv.URL+"/team/add", "POST", teamPayload)

	// Создание PR
	prPayload := map[string]string{
		"pull_request_id":   "pr-deactivate",
		"pull_request_name": "Test",
		"author_id":         "lead-1",
	}
	resp := makeRequest(t, srv.URL+"/pullRequest/create", "POST", prPayload)
	var prRespWrapper struct {
		PR models.PullRequest `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prRespWrapper); err != nil {
		t.Fatalf("Failed to decode PR response: %v, body: %s", err, readBody(t, resp))
	}
	prResp := prRespWrapper.PR

	if len(prResp.AssignedReviewers) == 0 {
		t.Fatal("Expected reviewers to be assigned")
	}

	// Деактивация большинства членов команды (оставляем member-3)
	deactivatePayload := map[string]interface{}{
		"team_name": "deactivate-team",
		"user_ids":  []string{"member-1", "member-2"},
	}
	resp = makeRequest(t, srv.URL+"/team/deactivateMembers", "POST", deactivatePayload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	// Создание нового PR (должен быть назначен только member-3)
	prPayload2 := map[string]string{
		"pull_request_id":   "pr-deactivate-2",
		"pull_request_name": "Test2",
		"author_id":         "lead-1",
	}
	resp = makeRequest(t, srv.URL+"/pullRequest/create", "POST", prPayload2)
	var prRespWrapper2 struct {
		PR models.PullRequest `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prRespWrapper2); err != nil {
		t.Fatalf("Failed to decode PR response: %v, body: %s", err, readBody(t, resp))
	}
	prResp2 := prRespWrapper2.PR

	if len(prResp2.AssignedReviewers) != 1 {
		t.Errorf("Expected 1 reviewer (member-3), got %d", len(prResp2.AssignedReviewers))
	} else if prResp2.AssignedReviewers[0] != "member-3" {
		t.Errorf("Expected member-3, got %s", prResp2.AssignedReviewers[0])
	}
}

func TestE2E_IdempotentMerge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	cleanupDB(t, db)

	srv := setupTestServer(db)
	defer srv.Close()

	// Создание команды и PR
	teamPayload := map[string]interface{}{
		"team_name": "idempotent-team",
		"members": []map[string]interface{}{
			{"user_id": "idem-1", "username": "User1", "is_active": true},
		},
	}
	makeRequest(t, srv.URL+"/team/add", "POST", teamPayload)

	prPayload := map[string]string{
		"pull_request_id":   "pr-idempotent",
		"pull_request_name": "Test",
		"author_id":         "idem-1",
	}
	makeRequest(t, srv.URL+"/pullRequest/create", "POST", prPayload)

	// Первый merge
	mergePayload := map[string]string{"pull_request_id": "pr-idempotent"}
	resp1 := makeRequest(t, srv.URL+"/pullRequest/merge", "POST", mergePayload)
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("First merge failed: %d", resp1.StatusCode)
	}

	var prRespWrapper1 struct {
		PR models.PullRequest `json:"pr"`
	}
	if err := json.NewDecoder(resp1.Body).Decode(&prRespWrapper1); err != nil {
		t.Fatalf("Failed to decode first merge PR response: %v, body: %s", err, readBody(t, resp1))
	}
	pr1 := prRespWrapper1.PR

	// Второй merge (идемпотентность)
	resp2 := makeRequest(t, srv.URL+"/pullRequest/merge", "POST", mergePayload)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("Second merge failed: %d", resp2.StatusCode)
	}

	var prRespWrapper2 struct {
		PR models.PullRequest `json:"pr"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&prRespWrapper2); err != nil {
		t.Fatalf("Failed to decode second merge PR response: %v, body: %s", err, readBody(t, resp2))
	}
	pr2 := prRespWrapper2.PR

	if pr1.Status != pr2.Status || pr1.PullRequestID != pr2.PullRequestID {
		t.Error("Merge is not idempotent")
	}
}

func makeRequest(t *testing.T, url, method string, payload interface{}) *http.Response {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
