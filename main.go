package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin/testdata/protoexample"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

type User struct {
	Name     string `json:"name" binding:"required"`
	Age      int    `json:"age"`
	Password string `json:"password" binding:"required"`
}

// sign
var secret = []byte("HelloWorld")

// payload信息
type Claims struct {
	Name     string `json:"user_name"`
	Password string `json:"password"`
	jwt.StandardClaims
}

//生成token
func GenerateToken(user *User) (string, error) {

	now := time.Now()
	// 过期时间
	expire := now.Add(3 * time.Minute)

	claims := Claims{
		Name:     user.Name,
		Password: user.Password,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expire.Unix(),
			Issuer:    "HelloWorld",
		},
	}
	// HS256加密
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 生成签名字符串，获取完整加密token
	return token.SignedString(secret)

}

// 解析token
func ParseToken(token string) (*Claims, error) {

	// 解析token，func(token *jwt.Token)获取签名
	claims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if claims != nil {
		if c, ok := claims.Claims.(*Claims); ok && claims.Valid { // Valid校验exp、iat、nbf
			return c, nil
		}
	}
	return nil, err
}

// 校验中间件
func AuthMid() gin.HandlerFunc {

	return func(c *gin.Context) {

		token := c.GetHeader("token")
		log.Printf("Token from header: %s \n", token)

		url := c.Request.URL
		log.Printf("Path: %s \t Rawpath: %s \n", url.Path, url.RawPath)
		code := http.StatusOK
		message := ""
		if token == "" {
			code = http.StatusUnauthorized
			message = "Token is empty."
		} else {
			if parseToken, err := ParseToken(token); err != nil {
				code = http.StatusUnauthorized
				message = "Token is invalid."
			} else if time.Now().Unix() > parseToken.ExpiresAt {
				code = http.StatusInternalServerError
				message = "Token is expired."
			}
		}

		if code != http.StatusOK {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
				"date":    time.Now().Format("2006-01-02 15:04:05"),
			})
			c.Abort()
			return
		}
		c.Next()
	}

}

// @title Test Swagger
// @version 1.0
// @description Just for study

func main() {

	// 日志同时写到文件和控制台
	file, _ := os.Create("./log/gin.log")
	gin.DefaultWriter = io.MultiWriter(file, os.Stdout)

	// 强制使用颜色
	gin.ForceConsoleColor()

	// Default使用Logger和Recovery中间件
	//app := gin.Default()

	// debug
	gin.SetMode(gin.DebugMode)
	// 使用其他中间件
	app := gin.New()

	// 日志中间件
	app.Use(gin.Logger())

	// 异常中间件，panic写入500错误码
	app.Use(gin.Recovery())

	app.LoadHTMLGlob("template/*")
	app.Static("/static", "./static")

	// json
	app.GET("/welcome", AuthMid(), func(context *gin.Context) {
		context.JSON(200, User{Name: "Hello", Age: 24})
	})

	// asciiJson
	app.GET("/asciiJson", func(context *gin.Context) {
		data := map[string]interface{}{
			"one": "This is One, 这是中文",
			"two": "This is two",
		}
		context.AsciiJSON(http.StatusOK, data)
		sprints := fmt.Sprintf("Hello%s", "hello")
		println(sprints)

	})

	// protobuf
	app.GET("/buf", func(context *gin.Context) {
		reps := []int64{int64(1), int64(2)}
		label := "test"
		data := &protoexample.Test{
			Label: &label,
			Reps:  reps,
		}
		context.ProtoBuf(http.StatusOK, data)
	})

	// html template
	app.GET("/html", func(context *gin.Context) {
		context.HTML(http.StatusOK, "test.tmpl", &gin.H{
			"title": "this is title",
		})
	})

	// 单个文件上传
	app.POST("/singlePost", func(context *gin.Context) {
		file, _ := context.FormFile("file")
		fmt.Println(file.Filename)
		if err := context.SaveUploadedFile(file, "./static/"+file.Filename); err != nil {
			context.JSON(http.StatusInternalServerError, err)
			return
		}
		context.JSON(http.StatusOK, gin.H{"fileName": file.Filename})
	})

	// struct 绑定
	app.GET("/bind", func(c *gin.Context) {
		user := User{}

		if err := c.ShouldBindJSON(&user); err != nil {
			panic(err)
		}

		c.JSON(http.StatusOK, user)
	})
	// generate user map
	users := make(map[string]User)
	users["one"] = User{
		Name:     "one",
		Age:      18,
		Password: "123456",
	}
	users["two"] = User{
		Name:     "two",
		Age:      18,
		Password: "123456",
	}
	// login生成token
	app.POST("/login",
		// @Accept json
		// @Produce json
		// @Param name query string true
		// @Param password query string true
		// @Router /login [post]
		func(c *gin.Context) {
			user := User{}
			err := c.BindJSON(&user)
			if err != nil {
				panic(err)
			}

			u := users[user.Name]
			if u.Name == "" {
				panic("user is not exit.")
			}
			if u.Password != user.Password {
				panic("Password invalid.")
			}

			if token, err := GenerateToken(&u); err != nil {
				panic(err)
			} else {
				c.JSON(http.StatusOK, gin.H{
					"code":  http.StatusOK,
					"token": token,
					"date":  time.Now().Format("2006/01/02 15:04:05"),
				})
			}
			panic("Failed to get token.")
		})

	// swagger
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	server := &http.Server{
		Addr:    ":8888",
		Handler: app,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown...")

	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Shutdown...", err)
	}
	log.Println("Server exit...")

}
