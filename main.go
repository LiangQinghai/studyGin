package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type User struct {
	Name string
	Age  int
}

func main() {

	app := gin.Default()

	app.GET("/welcome", func(context *gin.Context) {
		context.JSON(200, User{Name: "Hello", Age: 24})
	})

	app.GET("/asciiJson", func(context *gin.Context) {
		data := map[string]interface{}{
			"one": "This is One, 这是中文",
			"two": "This is two",
		}
		context.AsciiJSON(http.StatusOK, data)
		sprints := fmt.Sprintf("Hello%s", "hello")
		println(sprints)

	})

	err := app.Run(":8888")

	if err != nil {
		fmt.Println(err)
	}

}
