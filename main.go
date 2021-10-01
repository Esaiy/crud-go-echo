package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"

	_ "github.com/go-sql-driver/mysql"

	"github.com/labstack/echo"
)

type (
	User struct {
		Id       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Roles    int    `json:"roles"`
	}
	Handler struct {
		DB *sql.DB
	}
)

func main() {
	e := echo.New()
	ip, _ := exec.Command("wsl hostname -I").Output()

	db, err := sql.Open("mysql", fmt.Sprintf("root:toor@tcp(%v:6033)/task1", ip))
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("db is connected")
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println(err.Error())
	}

	h := &Handler{DB: db}

	e.GET("/user", h.GetUserList)
	e.GET("/user/:id", h.GetUserDetails)
	e.POST("/user", h.CreateUser)
	e.PUT("/user/:id", h.UpdateUser)
	e.DELETE("/user/:id", h.DeleteUser)
	e.Logger.Fatal(e.Start(":8000"))
}

func (h *Handler) GetUserList(c echo.Context) error {
	query := "SELECT * FROM users;"
	rows, err := h.DB.Query(query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var usr User
		if err := rows.Scan(&usr.Id, &usr.Username, &usr.Email, &usr.Roles); err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"message": "Data berhasil didapatkan",
					"data":    struct{}{},
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": "Terjadi kesalahan pada server",
				"error":   err.Error(),
			})
		}
		users = append(users, usr)
	}
	if err = rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Data berhasil didapatkan",
		"data":    users,
	})
}

func (h *Handler) GetUserDetails(c echo.Context) error {
	query := "SELECT * FROM users WHERE id = ?"
	userId, _ := strconv.Atoi(c.Param("id"))
	var user User
	if err := h.DB.QueryRow(query, userId).Scan(&user.Id, &user.Username, &user.Email, &user.Roles); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"message": "Data user tidak ditemukan",
				"data":    struct{}{},
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Data berhasil didapatkan",
		"data":    user,
	})
}

func (h *Handler) CreateUser(c echo.Context) error {
	username := c.FormValue("username")
	email := c.FormValue("email")

	rxEmail := regexp.MustCompile(`.+@.+\..+`)
	match := rxEmail.Match([]byte(email))
	if !match {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message": "Format email kurang tepat!",
		})
	}
	roles, err := strconv.Atoi(c.FormValue("roles"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message": `Field 'roles' harus berupa integer!`,
		})
	}

	query := "INSERT INTO users(username, email, roles) VALUES(?, ?, ?)"
	stmt, err := h.DB.Prepare(query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}
	defer stmt.Close()
	result, err := stmt.Exec(username, email, roles)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}
	insertedId, _ := result.LastInsertId()
	user := User{
		Id:       insertedId,
		Username: username,
		Email:    email,
		Roles:    roles,
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "User berhasil ditambahkan",
		"data":    user,
	})
}

func (h *Handler) UpdateUser(c echo.Context) error {
	changed := false
	query := "SELECT * FROM users WHERE id = ?"
	userId, _ := strconv.Atoi(c.Param("id"))
	var user User
	if err := h.DB.QueryRow(query, userId).Scan(&user.Id, &user.Username, &user.Email, &user.Roles); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "Data user tidak ditemukan",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}

	if username := c.FormValue("username"); username != "" {
		user.Username = username
		changed = true
	}
	if email := c.FormValue("email"); email != "" {
		rxEmail := regexp.MustCompile(`.+@.+\..+`)
		match := rxEmail.Match([]byte(email))
		if !match {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "Format email kurang tepat!",
			})
		}
		user.Email = email
		changed = true
	}
	if roles := c.FormValue("roles"); roles != "" {
		roles, err := strconv.Atoi(c.FormValue("roles"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": `Field 'roles' harus berupa integer!`,
			})
		}
		user.Roles = roles
		changed = true
	}

	if !changed {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message": `Tidak ada perubahan yang dilakukan terhadap user`,
		})
	}

	query2 := "UPDATE users set username = ?, email = ?, roles = ? where id = ?"
	_, err := h.DB.Exec(query2, user.Username, user.Email, user.Roles, user.Id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Data berhasil didapatkan",
		"data":    user,
	})

}

func (h *Handler) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	query := "DELETE FROM users where id = ?"
	result, err := h.DB.Exec(query, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Terjadi kesalahan pada server",
			"error":   err.Error(),
		})
	}

	if row, _ := result.RowsAffected(); row < 1 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "User tidak ditemukan",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User berhasil dihapus",
		"id":      id,
	})
}
