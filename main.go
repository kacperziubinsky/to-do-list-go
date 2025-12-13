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

var sessions = make(map[string]int) 

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
    UserID      int      `json:"user_id"`
}

type User struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Password string `json:"-"`
}

type RegisterRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Token    string `json:"token"`
    Username string `json:"username"`
    UserID   int    `json:"user_id"`
}

func generateToken() string {
    return fmt.Sprintf("token_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        token = strings.TrimPrefix(token, "Bearer ")
        
        userID, exists := sessions[token]
        if !exists {
            http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
            return
        }

        r.Header.Set("X-User-ID", strconv.Itoa(userID))

        next(w, r)
    }
}

func register(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    if req.Username == "" || req.Password == "" {
        http.Error(w, "Username and password required", http.StatusBadRequest)
        return
    }

    var exists int
    err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&exists)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    if exists > 0 {
        http.Error(w, "Username already exists", http.StatusConflict)
        return
    }

    result, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", req.Username, req.Password)
    if err != nil {
        log.Printf("User insert error: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    userID, _ := result.LastInsertId()

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message":  "User registered successfully",
        "user_id":  userID,
        "username": req.Username,
    })
}

func login(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    if req.Username == "" || req.Password == "" {
        http.Error(w, "Username and password required", http.StatusBadRequest)
        return
    }

    var user User
    err := db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", req.Username).
        Scan(&user.ID, &user.Username, &user.Password)
    if err == sql.ErrNoRows {
        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
        return
    }
    if err != nil {
        log.Printf("Login query error: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    if user.Password != req.Password {
        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
        return
    }

    token := generateToken()
    sessions[token] = user.ID

    response := LoginResponse{
        Token:    token,
        Username: user.Username,
        UserID:   user.ID,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getAllTasks(w http.ResponseWriter, r *http.Request) {
    userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

    rows, err := db.Query("SELECT id, name, description, status, date, user_id FROM tasks WHERE user_id = ?", userID)
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

        err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Status, &dateStr, &t.UserID)
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

    row := db.QueryRow("SELECT id, name, description, status, date, user_id FROM tasks WHERE id = ? AND user_id = ?", id, userID)

    var task Task
    var dateStr string

    err = row.Scan(&task.ID, &task.Name, &task.Description, &task.Status, &dateStr, &task.UserID)
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

    userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

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

func deleteTask(w http.ResponseWriter, r *http.Request) {
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
    idStr := parts[len(parts)-1]

    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid Task ID", http.StatusBadRequest)
        return
    }

    result, err := db.Exec("DELETE FROM tasks WHERE id = ? AND user_id = ?", id, userID)
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

func getTasksByStatus(status string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))

        rows, err := db.Query("SELECT id, name, description, status, date, user_id FROM tasks WHERE status = ? AND user_id = ?", status, userID)
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
            if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Status, &dateStr, &t.UserID); err != nil {
                log.Printf("Row scanning error (status): %v", err)
                continue
            }
            parsedTime, _ := time.Parse("2006-01-02", dateStr)
            t.Date = JSONTime(parsedTime)
            filteredTasks = append(filteredTasks, t)
        }

        if len(filteredTasks) == 0 {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode([]Task{})
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(filteredTasks)
    }
}

func updateTaskStatus(id int, status string, userID int) error {
    result, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ? AND user_id = ?", status, id, userID)
    if err != nil {
        return err
    }
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return fmt.Errorf("Task not found")
    }
    return nil
}

func makeStatusHandler(statusValue string) http.HandlerFunc {
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

        idStr := parts[len(parts)-1]
        id, err := strconv.Atoi(idStr)
        if err != nil {
            http.Error(w, "Invalid Task ID", http.StatusBadRequest)
            return
        }

        if err := updateTaskStatus(id, statusValue, userID); err != nil {
            http.Error(w, "Task not found", http.StatusNotFound)
            return
        }

        row := db.QueryRow("SELECT id, name, description, status, date, user_id FROM tasks WHERE id = ? AND user_id = ?", id, userID)
        var task Task
        var dateStr string
        if err := row.Scan(&task.ID, &task.Name, &task.Description, &task.Status, &dateStr, &task.UserID); err != nil {
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
    fmt.Fprintln(w, "Welcome to the Task Management API with Authentication")
    fmt.Fprintln(w, "\nAuthentication Endpoints:")
    fmt.Fprintln(w, "POST   /register (JSON: {\"username\": \"...\", \"password\": \"...\"})")
    fmt.Fprintln(w, "POST   /login (JSON: {\"username\": \"...\", \"password\": \"...\"})")
    fmt.Fprintln(w, "\nTask Endpoints (Require Authorization: Bearer <token>):")
    fmt.Fprintln(w, "GET    /tasks")
    fmt.Fprintln(w, "GET    /tasks/{id}")
    fmt.Fprintln(w, "POST   /tasks/create (JSON payload)")
    fmt.Fprintln(w, "DELETE /tasks/delete/{id}")
    fmt.Fprintln(w, "GET    /tasks/pending, /tasks/completed, /tasks/in-progress")
    fmt.Fprintln(w, "POST   /tasks/complete/{id}, /tasks/in-progress/{id}, /tasks/pending/{id}")
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

    usersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        password TEXT NOT NULL
    );`
    _, err = db.Exec(usersTable)
    if err != nil {
        log.Fatalf("Users table creation error: %s", err)
    }

    tasksTable := `
    CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        name TEXT,
        description TEXT,
        status TEXT,
        date TEXT,
        user_id INTEGER,
        FOREIGN KEY (user_id) REFERENCES users(id)
    );`
    _, err = db.Exec(tasksTable)
    if err != nil {
        log.Fatalf("Tasks table creation error: %s", err)
    }

    fmt.Println("Connected to SQLite database file tasks.db!")
    fmt.Println("Application ready to run. Listening on :8080")

    mux := http.NewServeMux()

    mux.Handle("/", &homeHandler{})
    mux.HandleFunc("/register", register)
    mux.HandleFunc("/login", login)

    mux.HandleFunc("/tasks", authMiddleware(getAllTasks))
    mux.HandleFunc("/tasks/", authMiddleware(getTask))
    mux.HandleFunc("/tasks/create", authMiddleware(createTask))
    mux.HandleFunc("/tasks/delete/", authMiddleware(deleteTask))

    mux.HandleFunc("/tasks/pending", authMiddleware(getTasksByStatus("Pending")))
    mux.HandleFunc("/tasks/completed", authMiddleware(getTasksByStatus("Completed")))
    mux.HandleFunc("/tasks/in-progress", authMiddleware(getTasksByStatus("In Progress")))

    mux.HandleFunc("/tasks/complete/", authMiddleware(makeStatusHandler("Completed")))
    mux.HandleFunc("/tasks/in-progress/", authMiddleware(makeStatusHandler("In Progress")))
    mux.HandleFunc("/tasks/pending/", authMiddleware(makeStatusHandler("Pending")))

    log.Fatal(http.ListenAndServe(":8080", mux))
}