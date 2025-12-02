package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
	"strings"
)

type Task struct {
	 ID   int
	 Name string
	 Description string
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

type homeHandler struct{}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome to the Task Management API")
}

func main() {
 tasks = append(tasks, Task{ID: 1, Name: "Sample Task", Description: "This is a sample task"})
 tasks = append(tasks, Task{ID: 2, Name: "Another Task", Description: "This is another task"})
 
 mux := http.NewServeMux()

 mux.Handle("/", &homeHandler{})
 mux.HandleFunc("/tasks", getAllTasks)
 mux.HandleFunc("/tasks/", getTask)
 mux.HandleFunc("/tasks/create", createTask)


 http.ListenAndServe(":8080", mux)
}
