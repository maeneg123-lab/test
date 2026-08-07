package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "time"
    "os"
    "log"
    "context"
    "strings"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"


    _ "github.com/lib/pq"
)

var jwtSecret = []byte("твой_секретный_ключ_для_jwt")

func generateToken(userID int) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(time.Hour * 24).Unix(),
    })
    return token.SignedString(jwtSecret)
}

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

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tokenString := r.Header.Get("Authorization")
        if tokenString == "" {
            http.Error(w, "Токен не предоставлен", http.StatusUnauthorized)
            return
        }

        // Убираем "Bearer " из строки
        tokenString = strings.TrimPrefix(tokenString, "Bearer ")

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("неверный метод подписи")
            }
            return jwtSecret, nil
        })

        if err != nil || !token.Valid {
            http.Error(w, "Неверный токен", http.StatusUnauthorized)
            return
        }

        // Извлекаем user_id из токена и добавляем в контекст запроса
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            userID := int(claims["user_id"].(float64))
            r = r.WithContext(context.WithValue(r.Context(), "user_id", userID))
        }

        next(w, r)
    }
}

func (n *Notes) saveNotes(title string, description string, status string, userID int) error{
    _,err := n.db.Exec("INSERT INTO notes_list (title, description,status, user_id) VALUES ($1,$2,$3,$4)", title, description,status, userID,)
    return err
}

func (n *Notes) new_note(w http.ResponseWriter,r *http.Request){
    userID, ok := r.Context().Value("user_id").(int)
    if !ok {
        http.Error(w, "User not authorized", http.StatusUnauthorized)
        return
    }
    value := r.URL.Query()
    title:=value.Get("title")
    description:=value.Get("description")
    status:=value.Get("status")

    err := n.saveNotes(title, description, status, userID)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w, "success!")
}

func (n *Notes) get_notes(w http.ResponseWriter, r *http.Request){
    userID, ok := r.Context().Value("user_id").(int)
    if !ok {
        http.Error(w, "User not authorized", http.StatusUnauthorized)
        return
    }
    rows,err:=n.db.Query(`SELECT id,title,description,status,created_at FROM notes_list WHERE user_id=$1;`, userID,)
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

        fmt.Fprintf(w, "id: %d, title: %v, description: %v, status: %v, created_at: %v\n", id,title,description,status,created_at.Format("2006-01-02 15:04:05"))
    }
}

func (n *Notes) get_note(w http.ResponseWriter, r *http.Request){
    userID, ok := r.Context().Value("user_id").(int)
    if !ok {
        http.Error(w, "User not authorized", http.StatusUnauthorized)
        return
    }
    idstr := r.URL.Query().Get("id")
    id,err:= strconv.ParseInt(idstr, 10,64)

    rows,err:=n.db.Query(`SELECT id,title,description,status,created_at FROM notes_list WHERE id=$1 AND user_id =$2;`, id, userID,)
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

        fmt.Fprintf(w, "id: %d, title: %v, description: %v, status: %v, created_at: %v\n", id,title,description,status,created_at.Format("2006-01-02 15:04:05"))
    }
}

func (n *Notes) del_note(w http.ResponseWriter, r *http.Request){
    userID, ok := r.Context().Value("user_id").(int)
    if !ok {
        http.Error(w, "User not authorized", http.StatusUnauthorized)
        return
    }
    idstr := r.URL.Query().Get("id")
    id,err:=strconv.ParseInt(idstr, 10,64)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }

    _, err= n.db.Query(`DELETE FROM notes_list WHERE id=$1 AND user_id=$2;`, id, userID,)

    if err!= nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }

    fmt.Fprintf(w, "success! deleted note: %d", id)
}

func (n *Notes) search_note(w http.ResponseWriter, r *http.Request){
    userID, ok := r.Context().Value("user_id").(int)
    if !ok {
        http.Error(w, "User not authorized", http.StatusUnauthorized)
        return
    }
    q := r.URL.Query().Get("q")

    rows, err:=n.db.Query(`SELECT id,title FROM notes_list WHERE user_id = $1 AND title ILIKE $2 OR description ILIKE $1`, userID, "%"+q+"%")

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
            fmt.Fprintf(w, "error: %v", err)
            return
        }
        fmt.Fprintf(w, "id: %d, title: %v", id, title)
    }
}

func (n *Notes) put_note(w http.ResponseWriter,r *http.Request){
    userID, ok := r.Context().Value("user_id").(int)
    if !ok {
        http.Error(w, "User not authorized", http.StatusUnauthorized)
        return
    }
    status := r.URL.Query().Get("status")
    idstr :=  r.URL.Query().Get("id")
    id, err:= strconv.ParseInt(idstr, 10,64)

    _, err= n.db.Query(`UPDATE notes_list SET status=$1 WHERE id=$2 AND user_id=$3`, status, id, userID)
    if err!=nil{
        fmt.Fprintf(w, "error: %v", err)
        return
    }
    fmt.Fprintf(w, "success! id: %d, status: %v", id, status)
}

func (n  *Notes) register(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("username")
    password := r.URL.Query().Get("password")

    if username == "" || password == "" {
        http.Error(w, "Username and password are required", http.StatusBadRequest)
        return
    }

    // Хешируем пароль
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, "Error hashing password", http.StatusInternalServerError)
        return
    }

    // Сохраняем в БД
    _, err = n.db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", username, hashedPassword)
    if err != nil {
        http.Error(w, "Username already exists", http.StatusConflict)
        return
    }

    fmt.Fprintf(w, "User registered successfully")
}

func (n *Notes) login(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("username")
    password := r.URL.Query().Get("password")

    if username == "" || password == "" {
        http.Error(w, "Username and password are required", http.StatusBadRequest)
        return
    }

    // Ищем пользователя в БД
    var userID int
    var hashedPassword string
    err := n.db.QueryRow("SELECT id, password FROM users WHERE username=$1", username).Scan(&userID, &hashedPassword)
    if err != nil {
        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
        return
    }

    // Проверяем пароль
    err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
    if err != nil {
        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
        return
    }

    // Генерируем JWT-токен
    token, err := generateToken(userID)
    if err != nil {
        http.Error(w, "Error generating token", http.StatusInternalServerError)
        return
    }

    // Отправляем токен в ответе
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func main(){
     // Проверка переменной окружения
    dbURL := os.Getenv("DATABASE_URL")
    fmt.Println("DATABASE_URL =", dbURL)
    if dbURL == "" {
        log.Fatal("DATABASE_URL не найдена")
    }
    server := NewServer()
    //createTableSQL := `
    //CREATE TABLE notes_list (
    //id  SERIAL PRIMARY KEY,
    //title TEXT NOT NULL,
    //description TEXT,
    //status TEXT DEFAULT 'new',
    //created_at TIMESTAMP DEFAULT NOW()
    //);`
    //_, err := server.db.Exec(createTableSQL)
    //if err != nil {
        //log.Fatal("Ошибка создания таблицы:", err)
   // }
    //fmt.Println("Таблица tasks_list проверена/создана")

        createTableSQL := `
    CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
    );`

    _, err := server.db.Exec(createTableSQL)
    if err != nil {
        log.Fatal("Ошибка создания таблицы:", err)
    }
    fmt.Println("Таблица tasks_list проверена/создана")


    //updateBook := `
    //ALTER TABLE notes_list ADD COLUMN user_id INT REFERENCES users(id);`

    //_, err = server.db.Exec(updateBook)
    //if err!=nil{
        //log.Fatal("ошибка обновления таблицы: ", err)
    //}
    //fmt.Println("таблица обновлена")
    
    http.HandleFunc("/register", server.register)  // без middleware
    http.HandleFunc("/login", server.login)        // без middleware
    http.HandleFunc("/notes", authMiddleware(server.get_notes))
    http.HandleFunc("/search", authMiddleware(server.search_note))
    http.HandleFunc("/note", authMiddleware(server.get_note))
    http.HandleFunc("/new_note", authMiddleware(server.new_note))
    http.HandleFunc("/put_note", authMiddleware(server.put_note))
    http.HandleFunc("/del_note", authMiddleware(server.del_note))
    fmt.Println("сервер запусщен на http://localhost:8080")
     port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    fmt.Println("Сервер запущен на порту", port)
    http.ListenAndServe(":"+port, nil)

}


