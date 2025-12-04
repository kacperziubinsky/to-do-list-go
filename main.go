package main

import (
    "encoding/json"
    "fmt"
    "math/rand"
    "net/http"
    "strconv"
    "strings"
    "time"
)

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

func randomDate() JSONTime {
    days := rand.Intn(30)
    d := time.Now().AddDate(0, 0, -days)
    return JSONTime(d)
}

type Task struct {
    ID          int      `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Status      string   `json:"status"`
    Date        JSONTime `json:"date"`
}

var tasks []Task

func getAllTasks(w http.ResponseWriter, r *http.Request) {
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

    for _, task := range tasks {
        if task.ID == id {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(task)
            return
        }
    }
    http.Error(w, "Task not found", http.StatusNotFound)
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

    newTask.ID = len(tasks) + 1
    tasks = append(tasks, newTask)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(newTask)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    idStr := strings.TrimPrefix(r.URL.Path, "/tasks/delete/")
    if idStr == "" {
        http.Error(w, "Task ID is required", http.StatusBadRequest)
        return
    }
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid Task ID", http.StatusBadRequest)
        return
    }
    for i, task := range tasks {
        if task.ID == id {
            tasks = append(tasks[:i], tasks[i+1:]...)
            w.WriteHeader(http.StatusNoContent)
            return
        }
    }
    http.Error(w, "Task not found", http.StatusNotFound)
}

func currentTasks(status string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var filtered []Task
        for _, task := range tasks {
            if task.Status == status {
                filtered = append(filtered, task)
            }
        }

        if len(filtered) == 0 {
            http.Error(w, "Task not found", http.StatusNotFound)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(filtered)
    })
}

func changeTaskStatus(id int, status string) error {
    for i, task := range tasks {
        if task.ID == id {
            tasks[i].Status = status
            return nil
        }
    }
    return fmt.Errorf("Task not found")
}

func markTaskCompleted(id int) error {
    return changeTaskStatus(id, "Completed")
}

func markTaskInProgress(id int) error {
    return changeTaskStatus(id, "In Progress")
}

func markTaskPending(id int) error {
    return changeTaskStatus(id, "Pending")
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

        for _, task := range tasks {
            if task.ID == id {
                w.Header().Set("Content-Type", "application/json")
                json.NewEncoder(w).Encode(task)
                return
            }
        }

        http.Error(w, "Task not found", http.StatusNotFound)
    }
}



type homeHandler struct{}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Welcome to the Task Management API")
}

func main() {
    rand.Seed(time.Now().UnixNano())

    fmt.Println("Starting server on :8080")

    tasks = append(tasks, Task{
        ID: 1, Name: "Sample Task", Description: "This is a sample task",
        Status: "Pending", Date: randomDate(),
    })

    tasks = append(tasks, Task{
        ID: 2, Name: "Another Task", Description: "This is another task",
        Status: "Pending", Date: randomDate(),
    })

    tasks = append(tasks, Task{
        ID: 3, Name: "Completed Task", Description: "This task is completed",
        Status: "Completed", Date: randomDate(),
    })

    tasks = append(tasks, Task{
        ID: 4, Name: "In Progress Task", Description: "This task is in progress",
        Status: "In Progress", Date: randomDate(),
    })

    tasks = append(tasks, Task{
        ID: 5, Name: "Final Task", Description: "This is the final task",
        Status: "Pending", Date: randomDate(),
    })

    mux := http.NewServeMux()

    mux.Handle("/", &homeHandler{})
    mux.HandleFunc("/tasks", getAllTasks)
    mux.HandleFunc("/tasks/", getTask)
    mux.HandleFunc("/tasks/create", createTask)
    mux.HandleFunc("/tasks/delete/", deleteTask)
    mux.Handle("/tasks/current", currentTasks("Pending"))
    mux.Handle("/tasks/completed", currentTasks("Completed"))
    mux.Handle("/tasks/in-progress", currentTasks("In Progress"))

    mux.HandleFunc("/tasks/complete/", makeStatusHandler(markTaskCompleted))
    mux.HandleFunc("/tasks/in-progress/", makeStatusHandler(markTaskInProgress))
    mux.HandleFunc("/tasks/pending/", makeStatusHandler(markTaskPending))



    http.ListenAndServe(":8080", mux)
}
