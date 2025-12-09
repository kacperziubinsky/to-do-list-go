package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "time"

    _ "modernc.org/sqlite"
)

var db *sql.DB

type JSONTime time.Time

func (jt *JSONTime) UnmarshalJSON(b []byte) error {
    s := strings.Trim(string(b), "\"")
    if s == "" {
        *jt = JSONTime(time.Now())
        return nil
    }

    t, err := time.Parse("2006-01-02", s)
    if err != nil {
        return err
    }
    *jt = JSONTime(t)
    return nil
}

func (jt JSONTime) MarshalJSON() ([]byte, error) {
    t := time.Time(jt)
    formatted := t.Format("2006-01-02")
    return []byte(fmt.Sprintf("\"%s\"", formatted)), nil
}

type Task struct {
    ID          int      `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Status      string   `json:"status"`
    Date        JSONTime `json:"date"`
}

func getAllTasks(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id, name, description, status, date FROM tasks")
    if err != nil {
        log.Printf("SELECT query error: %v", err)
        http.Error(w, "Database read error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var tasks []Task
    for rows.Next() {
        var t Task
        var dateStr string

        err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Status, &dateStr)
        if err != nil {
            log.Printf("Row scanning error: %v", err)
            continue
        }

        parsedTime, err := time.Parse("2006-01-02", dateStr)
        if err != nil {
            log.Printf("Date parsing error: %v", err)
            continue
        }
        t.Date = JSONTime(parsedTime)
        tasks = append(tasks, t)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tasks)
}

func getTask(w http.ResponseWriter, r *http.Request) {
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

    row := db.QueryRow("SELECT id, name, description, status, date FROM tasks WHERE id = ?", id)

    var task Task
    var dateStr string

    err = row.Scan(&task.ID, &task.Name, &task.Description, &task.Status, &dateStr)
    if err == sql.ErrNoRows {
        http.Error(w, "Task not found", http.StatusNotFound)
        return
    }
    if err != nil {
        log.Printf("Task scanning error: %v", err)
        http.Error(w, "Database read error", http.StatusInternalServerError)
        return
    }

    parsedTime, _ := time.Parse("2006-01-02", dateStr)
    task.Date = JSONTime(parsedTime)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(task)
}

func createTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var newTask Task
    err := json.NewDecoder(r.Body).Decode(&newTask)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    if time.Time(newTask.Date).IsZero() {
        newTask.Date = JSONTime(time.Now())
    }

    dateFormatted := time.Time(newTask.Date).Format("2006-01-02")

    result, err := db.Exec(
        "INSERT INTO tasks (name, description, status, date) VALUES (?, ?, ?, ?)",
        newTask.Name, newTask.Description, "Pending", dateFormatted,
    )
    if err != nil {
        log.Printf("Database insert error: %v", err)
        http.Error(w, "Database write error", http.StatusInternalServerError)
        return
    }

    id, _ := result.LastInsertId()
    newTask.ID = int(id)
    newTask.Status = "Pending"

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(newTask)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
    if len(parts) < 3 {
        http.Error(w, "Task ID is required", http.StatusBadRequest)
        return
    }
    idStr := parts[len(parts)-1]

    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid Task ID", http.StatusBadRequest)
        return
    }

    result, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
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

func getTasksByStatus(status string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT id, name, description, status, date FROM tasks WHERE status = ?", status)
        if err != nil {
            log.Printf("SELECT query error (status): %v", err)
            http.Error(w, "Database read error", http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var filteredTasks []Task
        for rows.Next() {
            var t Task
            var dateStr string
            if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Status, &dateStr); err != nil {
                log.Printf("Row scanning error (status): %v", err)
                continue
            }
            parsedTime, _ := time.Parse("2006-01-02", dateStr)
            t.Date = JSONTime(parsedTime)
            filteredTasks = append(filteredTasks, t)
        }

        if len(filteredTasks) == 0 {
            http.Error(w, fmt.Sprintf("No tasks found with status: %s", status), http.StatusNotFound)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(filteredTasks)
    })
}

func updateTaskStatus(id int, status string) error {
    result, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", status, id)
    if err != nil {
        return err
    }
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return fmt.Errorf("Task not found")
    }
    return nil
}

func setTaskCompleted(id int) error {
    return updateTaskStatus(id, "Completed")
}

func setTaskInProgress(id int) error {
    return updateTaskStatus(id, "In Progress")
}

func setTaskPending(id int) error {
    return updateTaskStatus(id, "Pending")
}

func makeStatusHandler(change func(int) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost && r.Method != http.MethodPatch {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
        if len(parts) < 3 {
            http.Error(w, "Task ID is required", http.StatusBadRequest)
            return
        }

        idStr := parts[len(parts)-1]
        id, err := strconv.Atoi(idStr)
        if err != nil {
            http.Error(w, "Invalid Task ID", http.StatusBadRequest)
            return
        }

        if err := change(id); err != nil {
            http.Error(w, "Task not found", http.StatusNotFound)
            return
        }

        row := db.QueryRow("SELECT id, name, description, status, date FROM tasks WHERE id = ?", id)
        var task Task
        var dateStr string
        if err := row.Scan(&task.ID, &task.Name, &task.Description, &task.Status, &dateStr); err != nil {
            http.Error(w, "Database read error after update", http.StatusInternalServerError)
            return
        }
        parsedTime, _ := time.Parse("2006-01-02", dateStr)
        task.Date = JSONTime(parsedTime)

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(task)
    }
}

type homeHandler struct{}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Welcome to the Task Management API (SQLite Edition)")
    fmt.Fprintln(w, "\nAvailable Endpoints:")
    fmt.Fprintln(w, "GET    /tasks")
    fmt.Fprintln(w, "GET    /tasks/{id}")
    fmt.Fprintln(w, "POST   /tasks/create (JSON payload)")
    fmt.Fprintln(w, "DELETE /tasks/delete/{id}")
    fmt.Fprintln(w, "GET    /tasks/pending, /tasks/completed, /tasks/in-progress (Filter by status)")
    fmt.Fprintln(w, "POST   /tasks/complete/{id}, /tasks/in-progress/{id}, /tasks/pending/{id} (Change status)")
}

func main() {
    var err error

    db, err = sql.Open("sqlite", "./tasks.db")
    if err != nil {
        log.Fatalf("Database open error: %v", err)
    }
    defer db.Close()

    if err = db.Ping(); err != nil {
        log.Fatalf("Database connection error: %v", err)
    }

    sqlStmt := `
    CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        name TEXT,
        description TEXT,
        status TEXT,
        date TEXT
    );`
    _, err = db.Exec(sqlStmt)
    if err != nil {
        log.Fatalf("SQLite table creation error: %s", err)
    }

    fmt.Println("Connected to SQLite database file tasks.db!")
    fmt.Println("Application ready to run. Listening on :8080")

    mux := http.NewServeMux()

    mux.Handle("/", &homeHandler{})
    mux.HandleFunc("/tasks", getAllTasks)
    mux.HandleFunc("/tasks/", getTask)
    mux.HandleFunc("/tasks/create", createTask)
    mux.HandleFunc("/tasks/delete/", deleteTask)

    mux.Handle("/tasks/pending", getTasksByStatus("Pending"))
    mux.Handle("/tasks/completed", getTasksByStatus("Completed"))
    mux.Handle("/tasks/in-progress", getTasksByStatus("In Progress"))

    mux.HandleFunc("/tasks/complete/", makeStatusHandler(setTaskCompleted))
    mux.HandleFunc("/tasks/in-progress/", makeStatusHandler(setTaskInProgress))
    mux.HandleFunc("/tasks/pending/", makeStatusHandler(setTaskPending))

    log.Fatal(http.ListenAndServe(":8080", mux))
}
