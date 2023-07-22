package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

var DB *sql.DB

func answerGET(c *gin.Context) {
	city := c.Query("city")

	var rows *sql.Rows
	var err error
	if city == "" {
		rows, err = DB.Query("SELECT * FROM answers;")
	} else {
		rows, err = DB.Query("SELECT * FROM answers WHERE city = $1;", city)
	}

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, make([]string, 0))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "DB query failure",
			"error":   err.Error(),
		})
		return
	}

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var name string
		var city string
		var age int
		var mail string
		var points int
		var submitDate string
		var question1Answer int
		var question2Answer int

		err = rows.Scan(&id, &name, &city, &age, &mail, &points, &submitDate, &question1Answer, &question2Answer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "DB row scan failure",
				"error":   err.Error(),
			})
			return
		}

		result = append(result, gin.H{
			"id":              id,
			"name":            name,
			"city":            city,
			"age":             age,
			"mail":            mail,
			"points":          points,
			"submitDate":      submitDate,
			"question1Answer": question1Answer,
			"question2Answer": question2Answer,
		})
	}

	c.JSON(http.StatusOK, result)
}

func answerPOST(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return
	}

	var jsonData map[string]interface{}
	json.Unmarshal(body, &jsonData)

	q1 := 0
	q2 := 0
	if jsonData["question_1_answer"] == true {
		q1 = 1
	}
	if jsonData["question_2_answer"] == true {
		q2 = 1
	}

	_, err = DB.Exec(
		"INSERT INTO answers(name,city,age,mail,points,submitDate,question1Answer,question2Answer) VALUES($1,$2,$3,$4,$5,$6,$7,$8);",
		jsonData["name"],
		jsonData["city"],
		jsonData["age"],
		jsonData["mail"],
		jsonData["points"],
		time.Now().Format("1999-01-08"),
		q1,
		q2,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "DB insert failure",
			"err":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}

func createTablesPOST(c *gin.Context) {
	sqlStmt := `
	CREATE TABLE answers (
		id SERIAL PRIMARY KEY,
		name TEXT,
		city TEXT,
		age INTEGER,
		mail TEXT,
		points INTEGER,
		submitDate DATE,
		question1Answer INTEGER,
		question2Answer INTEGER
	  );
	`

	_, err := DB.Exec(sqlStmt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "DB create table failure",
			"error":   err.Error(),
		})
		return
	}
}

func main() {
	var err error
	url := os.Getenv("DB_URL")
	connection, _ := pq.ParseURL(url)
	connection += " sslmode=require"

	// DB, err = sql.Open("postgres", "user="+os.Getenv("DB_USER")+" password="+os.Getenv("DB_PASSWORD")+"dbname="+os.Getenv("DB_NAME")+" sslmode=verify-full")
	DB, err = sql.Open("postgres", connection)
	if err != nil {
		panic(err)
	}
	DB.SetConnMaxLifetime(time.Minute * 3)
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(10)

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.POST("/send", answerPOST)
	router.GET("/answers", answerGET)

	router.POST("/create-tables", createTablesPOST)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Panicf("error: %s", err)
	}
}
