package main

import (
	"fmt"
	"log"
	"net/http"

	"moj_pierwszy_projekt/middleware"
	"moj_pierwszy_projekt/db"
	"moj_pierwszy_projekt/handlers"
)

type homeHandler struct{}

func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Task Management API")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Authentication:")
	fmt.Fprintln(w, "  POST   /register        {username, password}")
	fmt.Fprintln(w, "  POST   /login           {username, password}")
	fmt.Fprintln(w, "  POST   /logout          (requires token)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Tasks (all require: Authorization: Bearer <token>):")
	fmt.Fprintln(w, "  GET    /tasks           ?search=&sort=date|name|status&order=asc|desc&status=&date_from=&date_to=")
	fmt.Fprintln(w, "  GET    /tasks/{id}")
	fmt.Fprintln(w, "  POST   /tasks/create    {name, description, date}")
	fmt.Fprintln(w, "  PUT    /tasks/update/{id} {name, description, date}")
	fmt.Fprintln(w, "  DELETE /tasks/delete/{id}")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Status filters:")
	fmt.Fprintln(w, "  GET    /tasks/pending")
	fmt.Fprintln(w, "  GET    /tasks/completed")
	fmt.Fprintln(w, "  GET    /tasks/in-progress")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Status updates:")
	fmt.Fprintln(w, "  POST   /tasks/complete/{id}")
	fmt.Fprintln(w, "  POST   /tasks/in-progress/{id}")
	fmt.Fprintln(w, "  POST   /tasks/pending/{id}")
}

func main() {
	if err := db.Init("./tasks.db"); err != nil {
		log.Fatalf("DB init error: %v", err)
	}
	defer db.DB.Close()

	fmt.Println("Application ready. Listening on :8080")

	mux := http.NewServeMux()

	mux.Handle("/", &homeHandler{})
	mux.HandleFunc("/register", handlers.Register)
	mux.HandleFunc("/login", handlers.Login)
	mux.HandleFunc("/logout", middleware.Auth(handlers.Logout))

	mux.HandleFunc("/tasks", middleware.Auth(handlers.GetAllTasks))
	mux.HandleFunc("/tasks/create", middleware.Auth(handlers.CreateTask))
	mux.HandleFunc("/tasks/delete/", middleware.Auth(handlers.DeleteTask))
	mux.HandleFunc("/tasks/update/", middleware.Auth(handlers.UpdateTask))

	mux.HandleFunc("/tasks/pending", middleware.Auth(handlers.GetTasksByStatus("Pending")))
	mux.HandleFunc("/tasks/completed", middleware.Auth(handlers.GetTasksByStatus("Completed")))
	mux.HandleFunc("/tasks/in-progress", middleware.Auth(handlers.GetTasksByStatus("In Progress")))

	mux.HandleFunc("/tasks/complete/", middleware.Auth(handlers.MakeStatusHandler("Completed")))
	mux.HandleFunc("/tasks/in-progress/", middleware.Auth(handlers.MakeStatusHandler("In Progress")))
	mux.HandleFunc("/tasks/pending/", middleware.Auth(handlers.MakeStatusHandler("Pending")))

	mux.HandleFunc("/tasks/", middleware.Auth(handlers.GetTask))

	log.Fatal(http.ListenAndServe(":8080", mux))
}