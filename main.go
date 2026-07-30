package main

import (
    "database/sql"
    //"encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "time"
    "os"

    _ "github.com/lib/pq"
)


type Tasks struct{
    db *sql.DB
}

func NewServer() *Tasks {
    // Сначала пробуем взять строку из окружения
    connStr := os.Getenv("DATABASE_URL")
    // Если её нет — используем локальную для разработки
    if connStr == "" {
        connStr = "user=postgres password=36863686 dbname=work_db sslmode=disable"
    }
    
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        panic(err)
    }
    return &Tasks{db: db}
}

func (s *Tasks) saveTask(title string, description string, status string) error{
    _,err:= s.db.Exec(
        "INSERT INTO tasks_list (title, description, status) VALUES ($1,$2,$3)", title,description,status,
    )
    return err
}

func (s *Tasks) get_tasks(w http.ResponseWriter,r *http.Request){
    rows, err:= s.db.Query(`
        SELECT id,title,description,status,created_at FROM tasks_list
    `)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w, "tasks list\n\n")
    for rows.Next(){
        var id int
        var title string
        var description string
        var status string
        var created_at time.Time

        err:=rows.Scan(&id,&title,&description,&status,&created_at)
        if err!=nil{continue}
        fmt.Fprintf(w, "id: %d, title: %v, description: %v, status: %v, created_at: %v\n", id,title,description,status,created_at.Format("2006-01-02 15:04:05"))
    }
}

func (s *Tasks) get_task(w http.ResponseWriter,r *http.Request){
    idstr := r.URL.Query().Get("id")

    id, err:= strconv.ParseInt(idstr, 10, 64)
    if err!=nil{return}
    rows, err:= s.db.Query(`
        SELECT id,title,description,status,created_at FROM tasks_list WHERE id=$1;
    `, id)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w, "task: %d\n\n", id)
    for rows.Next(){
        var id int
        var title string
        var description string
        var status string
        var created_at time.Time

        err:=rows.Scan(&id,&title,&description,&status,&created_at)
        if err!=nil{continue}
        fmt.Fprintf(w, "id: %d, title: %v, description: %v, status: %v, created_at: %v\n", id,title,description,status,created_at.Format("2006-01-02 15:04:05"))
    }
}

func (s *Tasks) add_task(w http.ResponseWriter,r *http.Request){
    value := r.URL.Query()

    title := value.Get("title")
    description:= value.Get("description")
    status := value.Get("status")

    err := s.saveTask(title, description, status)
    if err!= nil{
        fmt.Fprintf(w, "error: %v", err)
    }

    fmt.Fprintf(w, "success!")
}

func (s *Tasks) put_task(w http.ResponseWriter, r *http.Request){
    value := r.URL.Query()

    idstr := value.Get("id")

    id, err := strconv.ParseInt(idstr, 10, 64)
    if err != nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    status := value.Get("status")

    _, err = s.db.Query(`
        UPDATE tasks_list SET status = $1 WHERE id=$2;
    `, status, id)

    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }

    fmt.Fprintf(w, "success: %v", id)
}

func (s *Tasks) del_task(w http.ResponseWriter,r *http.Request){
    value := r.URL.Query()

    idstr := value.Get("id")

    id,err := strconv.ParseInt(idstr, 10, 64)
    if err!=nil{return}

    _,err = s.db.Query(`
        DELETE FROM tasks_list WHERE id=$1;
    `, id)

    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }

    fmt.Fprintf(w, "success")
}



func main() {
    server := NewServer()
    
    // Создаём таблицу, если её нет
    createTableSQL := `
    CREATE TABLE IF NOT EXISTS tasks_list (
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        description TEXT,
        status TEXT DEFAULT 'pending',
        created_at TIMESTAMP DEFAULT NOW()
    );`
    
    _, err := server.db.Exec(createTableSQL)
    if err != nil {
        panic(err)
    }
    fmt.Println("Таблица 'tasks_list' создана/проверена")
    

    
    http.HandleFunc("/get_tasks", server.get_tasks)
    http.HandleFunc("/del_task", server.del_task)
    http.HandleFunc("/get_task", server.get_task)
    http.HandleFunc("/add_task", server.add_task)
    http.HandleFunc("/put_task", server.put_task)
    fmt.Println("Сервер запущен на http://localhost:8080")
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    fmt.Println("Сервер запущен на порту", port)
    http.ListenAndServe(":"+port, nil)
}



