package main

import (
    "database/sql"
    //"encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "time"
    "os"
    "log"


    _ "github.com/lib/pq"
)

type Notes struct{
    db *sql.DB
}

func NewServer() *Notes{
    // Сначала пробуем взять строку из окружения
    connStr := os.Getenv("DATABASE_URL")
    // Если её нет — используем локальную для разработки
    if connStr == "" {
        connStr = "user=postgres password=36863686 dbname=work_db sslmode=disable"
    }
    
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    return &Notes{db: db}

}

func (n *Notes) saveNotes(title string, description string, status string) error{
    _,err := n.db.Exec("INSERT INTO notes_list (title, description,status) VALUES ($1,$2,$3)", title, description,status,)
    return err
}

func (n *Notes) new_note(w http.ResponseWriter,r *http.Request){
    value := r.URL.Query()
    title:=value.Get("title")
    description:=value.Get("description")
    status:=value.Get("status")

    err := n.saveNotes(title, description, status)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w, "success!")
}

func (n *Notes) get_notes(w http.ResponseWriter, r *http.Request){
    rows,err:=n.db.Query(`SELECT id,title,description,status,created_at FROM notes_list;`)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w,"notes list:\n")
    for rows.Next(){
    var id int
        var title string
        var description string
        var status string
        var created_at time.Time

        err=rows.Scan(&id,&title,&description,&status,&created_at)

        fmt.Fprintf(w, "id: %d, title: %v, description: %v, status: %v, created_at: %v\n", id,title,description,status,created_at.Format("2006-04-24 12:11:00"))
    }
}

func (n *Notes) get_note(w http.ResponseWriter, r *http.Request){
    idstr := r.URL.Query().Get("id")
    id,err:= strconv.ParseInt(idstr, 10,64)

    rows,err:=n.db.Query(`SELECT id,title,description,status,created_at FROM notes_list WHERE id=$1;`, id,)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w,"note: %d\n", id)
    for rows.Next(){
    var id int
        var title string
        var description string
        var status string
        var created_at time.Time

        err=rows.Scan(&id,&title,&description,&status,&created_at)

        fmt.Fprintf(w, "id: %d, title: %v, description: %v, status: %v, created_at: %v\n", id,title,description,status,created_at.Format("2006-04-24 12:11:00"))
    }
}

func (n *Notes) del_note(w http.ResponseWriter, r *http.Request){
    idstr := r.URL.Query().Get("id")
    id,err:=strconv.ParseInt(idstr, 10,64)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }

    _, err= n.db.Query(`DELETE FROM notes_list WHERE id=$1;`, id,)

    if err!= nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }

    fmt.Fprintf(w, "success! deleted note: %d", id)
}

func (n *Notes) search_note(w http.ResponseWriter, r *http.Request){
    q := r.URL.Query().Get("q")

    rows, err:=n.db.Query(`SELECT id,title FROM notes_list WHERE title ILIKE $1 OR description ILIKE $1`, "%"+q+"%")

    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w, "result search for %v", q)
    for rows.Next(){
        var id int
        var title string

        err:=rows.Scan(&id, &title)
        if err!=nil{
            fmt.Fprintf(w, "error: ", err)
            return
        }
        fmt.Fprintf(w, "id: %d, title: %v", id, title)
    }
}

func (n *Notes) put_note(w http.ResponseWriter,r *http.Request){
    status := r.URL.Query().Get("status")
    idstr :=  r.URL.Query().Get("id")
    id, err:= strconv.ParseInt(idstr, 10,64)

    _, err= n.db.Query(`UPDATE notes_list SET status=$1 WHERE id=$2`, status, id)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w, "success! id: %d, status: %v", id, status)
}

func main(){
     // Проверка переменной окружения
    dbURL := os.Getenv("DATABASE_URL")
    fmt.Println("DATABASE_URL =", dbURL)
    if dbURL == "" {
        log.Fatal("DATABASE_URL не найдена")
    }
    server := NewServer()
    createTableSQL := `
    CREATE TABLE notes_list (
    id  SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'new',
    created_at TIMESTAMP DEFAULT NOW()
    );`
    _, err := server.db.Exec(createTableSQL)
    if err != nil {
        log.Fatal("Ошибка создания таблицы:", err)
    }
    fmt.Println("Таблица tasks_list проверена/создана")


    

    http.HandleFunc("/notes", server.get_notes)
    http.HandleFunc("/search", server.search_note)
    http.HandleFunc("/note", server.get_note)
    http.HandleFunc("/new_note", server.new_note)
    http.HandleFunc("/put_note", server.put_note)
    http.HandleFunc("/del_note", server.del_note)
    fmt.Println("сервер запусщен на http://localhost:8080")
     port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    fmt.Println("Сервер запущен на порту", port)
    http.ListenAndServe(":"+port, nil)

}


