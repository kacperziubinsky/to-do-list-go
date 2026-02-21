package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"moj_pierwszy_projekt/db"
	"moj_pierwszy_projekt/models"
)

// -------------------- GET /tasks --------------------
// Query params:
//   search=keyword        (searches name and description)
//   sort=date|name|status (default: date)
//   order=asc|desc        (default: asc)
//   status=Pending|Completed|In Progress
//   date_from=2006-01-02
//   date_to=2006-01-02

func GetAllTasks(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	params := parseQueryParams(r)

	query, args := buildTaskQuery(userID, params)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("SELECT query error: %v", err)
		http.Error(w, "Database read error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := scanTasks(rows)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func parseQueryParams(r *http.Request) models.TaskQueryParams {
	q := r.URL.Query()

	sortBy := q.Get("sort")
	if sortBy != "name" && sortBy != "status" && sortBy != "date" {
		sortBy = "date"
	}

	order := strings.ToUpper(q.Get("order"))
	if order != "DESC" {
		order = "ASC"
	}

	return models.TaskQueryParams{
		Search:   q.Get("search"),
		SortBy:   sortBy,
		Order:    order,
		Status:   q.Get("status"),
		DateFrom: q.Get("date_from"),
		DateTo:   q.Get("date_to"),
	}
}

func buildTaskQuery(userID int, p models.TaskQueryParams) (string, []interface{}) {
	query := "SELECT id, name, description, status, date, user_id FROM tasks WHERE user_id = ?"
	args := []interface{}{userID}

	if p.Search != "" {
		query += " AND (LOWER(name) LIKE ? OR LOWER(description) LIKE ?)"
		keyword := "%" + strings.ToLower(p.Search) + "%"
		args = append(args, keyword, keyword)
	}

	if p.Status != "" {
		query += " AND status = ?"
		args = append(args, p.Status)
	}

	if p.DateFrom != "" {
		query += " AND date >= ?"
		args = append(args, p.DateFrom)
	}

	if p.DateTo != "" {
		query += " AND date <= ?"
		args = append(args, p.DateTo)
	}

	// Safe to interpolate directly since sortBy is validated above
	query += fmt.Sprintf(" ORDER BY %s %s", p.SortBy, p.Order)

	return query, args
}

// -------------------- GET /tasks/{id} --------------------

func GetTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if idStr == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Task ID", http.StatusBadRequest)
		return
	}

	task, err := fetchTask(id, userID)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database read error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// -------------------- POST /tasks/create --------------------

func CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	var newTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(newTask.Name) == "" {
		http.Error(w, "Task name is required", http.StatusBadRequest)
		return
	}

	if time.Time(newTask.Date).IsZero() {
		newTask.Date = models.JSONTime(time.Now())
	}

	dateFormatted := time.Time(newTask.Date).Format("2006-01-02")

	result, err := db.DB.Exec(
		"INSERT INTO tasks (name, description, status, date, user_id) VALUES (?, ?, ?, ?, ?)",
		newTask.Name, newTask.Description, "Pending", dateFormatted, userID,
	)
	if err != nil {
		log.Printf("Database insert error: %v", err)
		http.Error(w, "Database write error", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	newTask.ID = int(id)
	newTask.Status = "Pending"
	newTask.UserID = userID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newTask)
}

// -------------------- PUT /tasks/update/{id} --------------------

func UpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "Invalid Task ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Task name is required", http.StatusBadRequest)
		return
	}

	dateFormatted := time.Time(req.Date).Format("2006-01-02")
	if time.Time(req.Date).IsZero() {
		dateFormatted = time.Now().Format("2006-01-02")
	}

	result, err := db.DB.Exec(
		"UPDATE tasks SET name = ?, description = ?, date = ? WHERE id = ? AND user_id = ?",
		req.Name, req.Description, dateFormatted, id, userID,
	)
	if err != nil {
		log.Printf("Update error: %v", err)
		http.Error(w, "Database update error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	task, err := fetchTask(id, userID)
	if err != nil {
		http.Error(w, "Database read error after update", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// -------------------- DELETE /tasks/delete/{id} --------------------

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "Invalid Task ID", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("DELETE FROM tasks WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		http.Error(w, "Database error during deletion", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -------------------- Status filter handlers --------------------

func GetTasksByStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

		rows, err := db.DB.Query(
			"SELECT id, name, description, status, date, user_id FROM tasks WHERE status = ? AND user_id = ?",
			status, userID,
		)
		if err != nil {
			log.Printf("SELECT query error (status): %v", err)
			http.Error(w, "Database read error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tasks := scanTasks(rows)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	}
}

// -------------------- Status update handlers --------------------

func MakeStatusHandler(statusValue string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPatch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 {
			http.Error(w, "Task ID is required", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			http.Error(w, "Invalid Task ID", http.StatusBadRequest)
			return
		}

		result, err := db.DB.Exec(
			"UPDATE tasks SET status = ? WHERE id = ? AND user_id = ?",
			statusValue, id, userID,
		)
		if err != nil || func() bool { ra, _ := result.RowsAffected(); return ra == 0 }() {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		task, err := fetchTask(id, userID)
		if err != nil {
			http.Error(w, "Database read error after update", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}
}

// -------------------- Helpers --------------------

func fetchTask(id, userID int) (models.Task, error) {
	row := db.DB.QueryRow(
		"SELECT id, name, description, status, date, user_id FROM tasks WHERE id = ? AND user_id = ?",
		id, userID,
	)
	var task models.Task
	var dateStr string
	err := row.Scan(&task.ID, &task.Name, &task.Description, &task.Status, &dateStr, &task.UserID)
	if err != nil {
		return task, err
	}
	parsedTime, _ := time.Parse("2006-01-02", dateStr)
	task.Date = models.JSONTime(parsedTime)
	return task, nil
}

func scanTasks(rows *sql.Rows) []models.Task {
	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var dateStr string
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Status, &dateStr, &t.UserID); err != nil {
			log.Printf("Row scanning error: %v", err)
			continue
		}
		parsedTime, _ := time.Parse("2006-01-02", dateStr)
		t.Date = models.JSONTime(parsedTime)
		tasks = append(tasks, t)
	}
	if tasks == nil {
		tasks = []models.Task{}
	}
	return tasks
}