package e2e_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL    = "http://app:8080"
	adminToken = "secret_admin_token_change_me"
)

// Client представляет HTTP клиент для тестов
type Client struct {
	baseURL    string
	httpClient *http.Client
	adminToken string
}

// NewClient создает новый тестовый клиент
func NewClient() *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		adminToken: adminToken,
	}
}

// doRequest выполняет HTTP запрос
func (c *Client) doRequest(method, path string, body interface{}, useAuth bool) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if useAuth {
		req.Header.Set("Authorization", "Bearer "+c.adminToken)
	}

	return c.httpClient.Do(req)
}

// waitForService ждет, пока сервис станет доступным
func waitForService(t *testing.T) {
	client := NewClient()
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		resp, err := client.httpClient.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Service did not become available in time")
}

// TestMain выполняется перед всеми тестами
func TestMain(m *testing.M) {
	// Ждем, пока сервис станет доступным
	time.Sleep(3 * time.Second)
	m.Run()
}

// TestHealthCheck проверяет health endpoint
func TestHealthCheck(t *testing.T) {
	waitForService(t)

	client := NewClient()
	resp, err := client.httpClient.Get(baseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

// TestCreateTeamFlow проверяет создание команды
func TestCreateTeamFlow(t *testing.T) {
	waitForService(t)
	client := NewClient()

	// Создаем команду
	teamReq := map[string]interface{}{
		"team_name": "engineering",
		"members": []map[string]interface{}{
			{
				"user_id":   "e2e_user_1",
				"username":  "Alice",
				"is_active": true,
			},
			{
				"user_id":   "e2e_user_2",
				"username":  "Bob",
				"is_active": true,
			},
			{
				"user_id":   "e2e_user_3",
				"username":  "Charlie",
				"is_active": true,
			},
		},
	}

	resp, err := client.doRequest("POST", "/team/add", teamReq, false)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	team := result["team"].(map[string]interface{})
	assert.Equal(t, "engineering", team["team_name"])
	assert.Len(t, team["members"], 3)

	// Попытка создать ту же команду снова - должна вернуть ошибку
	resp2, err := client.doRequest("POST", "/team/add", teamReq, false)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	var errResult map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&errResult)
	require.NoError(t, err)

	errDetail := errResult["error"].(map[string]interface{})
	assert.Equal(t, "TEAM_EXISTS", errDetail["code"])
}

// TestGetTeam проверяет получение команды
func TestGetTeam(t *testing.T) {
	waitForService(t)
	client := NewClient()

	// Получаем команду
	resp, err := client.httpClient.Get(baseURL + "/team/get?team_name=engineering")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "engineering", result["team_name"])
	assert.Len(t, result["members"], 3)

	// Попытка получить несуществующую команду
	resp2, err := client.httpClient.Get(baseURL + "/team/get?team_name=nonexistent")
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

// TestPullRequestFlow проверяет полный flow работы с PR
func TestPullRequestFlow(t *testing.T) {
	waitForService(t)
	client := NewClient()

	// 1. Создаем PR
	prReq := map[string]interface{}{
		"pull_request_id":   "e2e_pr_1",
		"pull_request_name": "Add feature",
		"author_id":         "e2e_user_1",
	}

	resp, err := client.doRequest("POST", "/pullRequest/create", prReq, false)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResult map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResult)
	require.NoError(t, err)

	pr := createResult["pr"].(map[string]interface{})
	assert.Equal(t, "e2e_pr_1", pr["pull_request_id"])
	assert.Equal(t, "Add feature", pr["pull_request_name"])
	assert.Equal(t, "e2e_user_1", pr["author_id"])
	assert.Equal(t, "OPEN", pr["status"])

	// Должны быть назначены до 2 ревьюверов
	reviewers := pr["assigned_reviewers"].([]interface{})
	assert.LessOrEqual(t, len(reviewers), 2)
	assert.GreaterOrEqual(t, len(reviewers), 1)

	// Проверяем, что автор не назначен ревьювером
	for _, reviewer := range reviewers {
		assert.NotEqual(t, "e2e_user_1", reviewer)
	}

	// 2. Проверяем получение PR для ревьювера
	firstReviewer := reviewers[0].(string)
	resp2, err := client.httpClient.Get(fmt.Sprintf("%s/users/getReview?user_id=%s", baseURL, firstReviewer))
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var reviewResult map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&reviewResult)
	require.NoError(t, err)

	assert.Equal(t, firstReviewer, reviewResult["user_id"])
	prs := reviewResult["pull_requests"].([]interface{})
	assert.GreaterOrEqual(t, len(prs), 1)

	// Проверяем, что наш PR в списке
	foundPR := false
	for _, prItem := range prs {
		prMap := prItem.(map[string]interface{})
		if prMap["pull_request_id"] == "e2e_pr_1" {
			foundPR = true
			break
		}
	}
	assert.True(t, foundPR, "PR должен быть в списке ревьюверов")

	// 3. Переназначаем ревьювера
	if len(reviewers) > 0 {
		reassignReq := map[string]interface{}{
			"pull_request_id": "e2e_pr_1",
			"old_user_id":     reviewers[0],
		}

		resp3, err := client.doRequest("POST", "/pullRequest/reassign", reassignReq, false)
		require.NoError(t, err)
		defer resp3.Body.Close()

		// Может быть 200 (успех) или 409 (нет кандидатов)
		assert.Contains(t, []int{http.StatusOK, http.StatusConflict}, resp3.StatusCode)

		if resp3.StatusCode == http.StatusOK {
			var reassignResult map[string]interface{}
			err = json.NewDecoder(resp3.Body).Decode(&reassignResult)
			require.NoError(t, err)

			newPR := reassignResult["pr"].(map[string]interface{})
			newReviewers := newPR["assigned_reviewers"].([]interface{})

			// Старый ревьювер не должен быть в списке
			for _, reviewer := range newReviewers {
				assert.NotEqual(t, reviewers[0], reviewer)
			}

			assert.NotEmpty(t, reassignResult["replaced_by"])
		}
	}

	// 4. Мёржим PR
	mergeReq := map[string]interface{}{
		"pull_request_id": "e2e_pr_1",
	}

	resp4, err := client.doRequest("POST", "/pullRequest/merge", mergeReq, false)
	require.NoError(t, err)
	defer resp4.Body.Close()

	assert.Equal(t, http.StatusOK, resp4.StatusCode)

	var mergeResult map[string]interface{}
	err = json.NewDecoder(resp4.Body).Decode(&mergeResult)
	require.NoError(t, err)

	mergedPR := mergeResult["pr"].(map[string]interface{})
	assert.Equal(t, "MERGED", mergedPR["status"])
	assert.NotNil(t, mergedPR["mergedAt"])

	// 5. Проверяем идемпотентность merge
	resp5, err := client.doRequest("POST", "/pullRequest/merge", mergeReq, false)
	require.NoError(t, err)
	defer resp5.Body.Close()

	assert.Equal(t, http.StatusOK, resp5.StatusCode)

	// 6. Попытка переназначить после merge - должна вернуть ошибку
	if len(reviewers) > 0 {
		reassignReq := map[string]interface{}{
			"pull_request_id": "e2e_pr_1",
			"old_user_id":     reviewers[0],
		}

		resp6, err := client.doRequest("POST", "/pullRequest/reassign", reassignReq, false)
		require.NoError(t, err)
		defer resp6.Body.Close()

		assert.Equal(t, http.StatusConflict, resp6.StatusCode)

		var errResult map[string]interface{}
		err = json.NewDecoder(resp6.Body).Decode(&errResult)
		require.NoError(t, err)

		errDetail := errResult["error"].(map[string]interface{})
		assert.Equal(t, "PR_MERGED", errDetail["code"])
	}
}

// TestUserActivation проверяет изменение активности пользователя
func TestUserActivation(t *testing.T) {
	waitForService(t)
	client := NewClient()

	// 1. Деактивируем пользователя с admin токеном
	setActiveReq := map[string]interface{}{
		"user_id":   "e2e_user_3",
		"is_active": false,
	}

	resp, err := client.doRequest("POST", "/users/setIsActive", setActiveReq, true)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	user := result["user"].(map[string]interface{})
	assert.Equal(t, "e2e_user_3", user["user_id"])
	assert.False(t, user["is_active"].(bool))

	// 2. Попытка без токена - должна вернуть 401
	resp2, err := client.doRequest("POST", "/users/setIsActive", setActiveReq, false)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp2.StatusCode)

	// 3. Активируем обратно
	setActiveReq["is_active"] = true
	resp3, err := client.doRequest("POST", "/users/setIsActive", setActiveReq, true)
	require.NoError(t, err)
	defer resp3.Body.Close()

	assert.Equal(t, http.StatusOK, resp3.StatusCode)
}

// TestPRWithInactiveUsers проверяет, что неактивные пользователи не назначаются
func TestPRWithInactiveUsers(t *testing.T) {
	waitForService(t)
	client := NewClient()

	// Создаем новую команду с одним активным пользователем
	teamReq := map[string]interface{}{
		"team_name": "small_team",
		"members": []map[string]interface{}{
			{
				"user_id":   "e2e_small_1",
				"username":  "David",
				"is_active": true,
			},
			{
				"user_id":   "e2e_small_2",
				"username":  "Eve",
				"is_active": false,
			},
		},
	}

	resp, err := client.doRequest("POST", "/team/add", teamReq, false)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Создаем PR от активного пользователя
	prReq := map[string]interface{}{
		"pull_request_id":   "e2e_pr_small",
		"pull_request_name": "Small team PR",
		"author_id":         "e2e_small_1",
	}

	resp2, err := client.doRequest("POST", "/pullRequest/create", prReq, false)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp2.Body).Decode(&result)
	require.NoError(t, err)

	pr := result["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})

	// Не должно быть ревьюверов, т.к. единственный другой член команды неактивен
	assert.Empty(t, reviewers)
}

// TestErrorCases проверяет различные error cases
func TestErrorCases(t *testing.T) {
	waitForService(t)
	client := NewClient()

	t.Run("Create PR with non-existent author", func(t *testing.T) {
		prReq := map[string]interface{}{
			"pull_request_id":   "e2e_pr_error_1",
			"pull_request_name": "Error PR",
			"author_id":         "nonexistent_user",
		}

		resp, err := client.doRequest("POST", "/pullRequest/create", prReq, false)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		errDetail := result["error"].(map[string]interface{})
		assert.Equal(t, "NOT_FOUND", errDetail["code"])
	})

	t.Run("Duplicate PR creation", func(t *testing.T) {
		prReq := map[string]interface{}{
			"pull_request_id":   "e2e_pr_1",
			"pull_request_name": "Duplicate PR",
			"author_id":         "e2e_user_1",
		}

		resp, err := client.doRequest("POST", "/pullRequest/create", prReq, false)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		errDetail := result["error"].(map[string]interface{})
		assert.Equal(t, "PR_EXISTS", errDetail["code"])
	})

	t.Run("Reassign non-assigned reviewer", func(t *testing.T) {
		// Создаём новый открытый PR для этого теста
		prReq := map[string]interface{}{
			"pull_request_id":   "e2e_pr_error_2",
			"pull_request_name": "Test PR for reassign error",
			"author_id":         "e2e_user_1",
		}

		respCreate, err := client.doRequest("POST", "/pullRequest/create", prReq, false)
		require.NoError(t, err)
		defer respCreate.Body.Close()
		assert.Equal(t, http.StatusCreated, respCreate.StatusCode)

		// Пытаемся переназначить пользователя, который НЕ является ревьювером
		reassignReq := map[string]interface{}{
			"pull_request_id": "e2e_pr_error_2",
			"old_user_id":     "e2e_user_1",
		}

		resp, err := client.doRequest("POST", "/pullRequest/reassign", reassignReq, false)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		errDetail := result["error"].(map[string]interface{})
		assert.Equal(t, "NOT_ASSIGNED", errDetail["code"])
	})

	t.Run("Get reviews for non-existent user", func(t *testing.T) {
		resp, err := client.httpClient.Get(baseURL + "/users/getReview?user_id=nonexistent")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestStatistics(t *testing.T) {
	waitForService(t)
	client := NewClient()

	resp, err := client.httpClient.Get(baseURL + "/statistics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&stats)
	require.NoError(t, err)

	assert.Contains(t, stats, "total_prs")
	assert.Contains(t, stats, "open_prs")
	assert.Contains(t, stats, "merged_prs")
	assert.Contains(t, stats, "assignments_by_user")
	assert.Contains(t, stats, "assignments_by_pr")
	assert.Contains(t, stats, "total_teams")
	assert.Contains(t, stats, "total_users")
	assert.Contains(t, stats, "active_users")

	assert.IsType(t, float64(0), stats["total_prs"])
	assert.IsType(t, map[string]interface{}{}, stats["assignments_by_user"])
	assert.IsType(t, map[string]interface{}{}, stats["assignments_by_pr"])

	totalPRs := stats["total_prs"].(float64)
	assert.Greater(t, totalPRs, float64(0), "Should have PRs from previous tests")

	totalTeams := stats["total_teams"].(float64)
	assert.Greater(t, totalTeams, float64(0), "Should have teams from previous tests")

	totalUsers := stats["total_users"].(float64)
	assert.Greater(t, totalUsers, float64(0), "Should have users from previous tests")

	t.Logf("Statistics: %+v", stats)
}

// TestTeamDeactivation проверяет массовую деактивацию команды
func TestTeamDeactivation(t *testing.T) {
	waitForService(t)
	client := NewClient()

	// 1. Создаём команду с несколькими пользователями
	teamReq := map[string]interface{}{
		"team_name": "deactivate_team",
		"members": []map[string]interface{}{
			{
				"user_id":   "deact_user1",
				"username":  "DeactUser1",
				"is_active": true,
			},
			{
				"user_id":   "deact_user2",
				"username":  "DeactUser2",
				"is_active": true,
			},
			{
				"user_id":   "deact_user3",
				"username":  "DeactUser3",
				"is_active": true,
			},
		},
	}

	resp, err := client.doRequest("POST", "/team/add", teamReq, false)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// 2. Создаём несколько открытых PR для этих пользователей
	// PR от deact_user1, ревьюверы: deact_user2, deact_user3
	prReq1 := map[string]interface{}{
		"pull_request_id":   "deact_pr1",
		"pull_request_name": "PR for deactivation test",
		"author_id":         "deact_user1",
	}

	resp2, err := client.doRequest("POST", "/pullRequest/create", prReq1, false)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	// 3. Массовая деактивация команды
	deactivateReq := map[string]interface{}{
		"team_name": "deactivate_team",
	}

	startTime := time.Now()
	resp3, err := client.doRequest("POST", "/team/deactivateMembers", deactivateReq, true)
	duration := time.Since(startTime)
	require.NoError(t, err)
	defer resp3.Body.Close()

	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	var deactivateResult map[string]interface{}
	err = json.NewDecoder(resp3.Body).Decode(&deactivateResult)
	require.NoError(t, err)

	// Проверяем результат
	assert.Equal(t, float64(3), deactivateResult["deactivated_count"]) // 3 пользователя
	assert.NotNil(t, deactivateResult["reassigned_prs"])
	assert.NotNil(t, deactivateResult["user_ids"])

	// 4. Проверяем производительность
	assert.Less(t, duration.Milliseconds(), int64(100),
		"Deactivation took %v ms, expected < 100 ms", duration.Milliseconds())

	// 5. Проверяем, что пользователи деактивированы
	resp4, err := client.httpClient.Get(baseURL + "/team/get?team_name=deactivate_team")
	require.NoError(t, err)
	defer resp4.Body.Close()

	var teamResult map[string]interface{}
	err = json.NewDecoder(resp4.Body).Decode(&teamResult)
	require.NoError(t, err)

	members := teamResult["members"].([]interface{})
	for _, member := range members {
		m := member.(map[string]interface{})
		assert.False(t, m["is_active"].(bool), "User %s should be inactive", m["user_id"])
	}

}
