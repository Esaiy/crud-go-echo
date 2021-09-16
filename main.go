package main

import (
	"database/sql"
	"fmt"
	"net/http"
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

type Cat struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func main() {
	e := echo.New()

	db, err := sql.Open("mysql", "root:toor@tcp(172.17.16.1:6033)/task1")
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
		fmt.Println(err.Error())
		return err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var usr User
		if err := rows.Scan(&usr.Id, &usr.Username, &usr.Email, &usr.Roles); err != nil {
			fmt.Println(err.Error())
			return err
		}
		users = append(users, usr)
	}
	if err = rows.Err(); err != nil {
		fmt.Println(err.Error())
		return err
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handler) GetUserDetails(c echo.Context) error {
	query := "SELECT * FROM users WHERE id = ?"
	userId, _ := strconv.Atoi(c.Param("id"))
	var user User
	if err := h.DB.QueryRow(query, userId).Scan(&user.Id, &user.Username, &user.Email, &user.Roles); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, nil)
		}
		fmt.Println(err.Error())
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (h *Handler) CreateUser(c echo.Context) error {
	username := c.FormValue("username")
	email := c.FormValue("email")
	roles, _ := strconv.Atoi(c.FormValue("roles"))

	query := "INSERT INTO users(username, email, roles) VALUES(?, ?, ?)"
	stmt, err := h.DB.Prepare(query)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	defer stmt.Close()
	result, err2 := stmt.Exec(username, email, roles)
	if err2 != nil {
		fmt.Println(err2.Error())
		return err2
	}
	insertedId, _ := result.LastInsertId()
	user := User{
		Id:       insertedId,
		Username: username,
		Email:    email,
		Roles:    roles,
	}
	return c.JSON(http.StatusCreated, user)
}

func (h *Handler) UpdateUser(c echo.Context) error {
	query := "SELECT * FROM users WHERE id = ?"
	userId, _ := strconv.Atoi(c.Param("id"))
	var user User
	if err := h.DB.QueryRow(query, userId).Scan(&user.Id, &user.Username, &user.Email, &user.Roles); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "User not Found",
			})
		}
		fmt.Println(err.Error())
		return err
	}

	if username := c.FormValue("username"); username != "" {
		user.Username = username
	}
	if email := c.FormValue("email"); email != "" {
		user.Email = email
	}
	if roles, err := strconv.Atoi(c.FormValue("roles")); err == nil {
		user.Roles = roles
	}

	query2 := "UPDATE users set username = ?, email = ?, roles = ? where id = ?"
	_, err := h.DB.Exec(query2, user.Username, user.Email, user.Roles, user.Id)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return c.JSON(http.StatusOK, user)

}

func (h *Handler) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	query := "DELETE FROM users where id = ?"
	result, err := h.DB.Exec(query, id)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	if row, _ := result.RowsAffected(); row < 1 {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "User not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User deleted successfully",
		"id":      id,
	})
}
