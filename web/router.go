// Package web
// @Description: 封装了所以web相关的内容
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/legolasljl/studyclaw/conf"
	"github.com/legolasljl/studyclaw/utils"
)

// 将静态文件嵌入到可执行程序中来
//go:embed studyclaw/build
var static embed.FS

var frontendBuild = mustFrontendBuildFS()

func mustFrontendBuildFS() fs.FS {
	buildFS, err := fs.Sub(static, "studyclaw/build")
	if err != nil {
		panic(err)
	}
	return buildFS
}

func isFrontendRedirectPath(path string) bool {
	switch path {
	case "/", "/studyclaw", "/studyclaw/index.html":
		return true
	default:
		return false
	}
}

func redirectToStableFrontendEntry(ctx *gin.Context) {
	target := "/studyclaw/"
	if rawQuery := ctx.Request.URL.RawQuery; rawQuery != "" {
		target += "?" + rawQuery
	}
	ctx.Redirect(http.StatusTemporaryRedirect, target)
}

// RouterInit
// @Description:
// @return *gin.Engine
func RouterInit() *gin.Engine {
	router := gin.Default()
	router.RemoveExtraSlash = true
	router.RedirectTrailingSlash = false
	router.Use(cors())
	router.Use(gzip.Gzip(1, gzip.WithExcludedExtensions([]string{"js", "css", "map", "png", "ico"})))
	router.Use(func(ctx *gin.Context) {
		if ctx.Request.Method == http.MethodGet || ctx.Request.Method == http.MethodHead {
			if isFrontendRedirectPath(ctx.Request.URL.Path) {
				redirectToStableFrontendEntry(ctx)
				ctx.Abort()
				return
			}
		}
		ctx.Next()
	})

	// 挂载前端静态文件，统一使用 /studyclaw/ 作为正式入口。
	router.StaticFS("/studyclaw", http.FS(frontendBuild))

	router.GET("/about", func(context *gin.Context) {
		context.JSON(200, Resp{
			Code:    200,
			Message: "",
			Data:    utils.GetAbout(),
			Success: true,
			Error:   "",
		})
	})

	router.POST("/restart", check(), func(ctx *gin.Context) {
		if ctx.GetInt("level") == 1 {
			ctx.JSON(200, Resp{
				Code:    200,
				Message: "",
				Data:    nil,
				Success: true,
				Error:   "",
			})
			utils.Restart()
		} else {
			ctx.JSON(200, Resp{
				Code:    401,
				Message: "",
				Data:    nil,
				Success: false,
				Error:   "",
			})
		}
	})

	// router.POST("/update", check(), func(ctx *gin.Context) {
	// 	if ctx.GetInt("level") == 1 {
	// 		update.SelfUpdate("", conf.GetVersion())
	// 		ctx.JSON(200, Resp{
	// 			Code:    200,
	// 			Message: "",
	// 			Data:    nil,
	// 			Success: true,
	// 			Error:   "",
	// 		})
	// 		utils.Restart()
	// 	} else {
	// 		ctx.JSON(200, Resp{
	// 			Code:    401,
	// 			Message: "",
	// 			Data:    nil,
	// 			Success: false,
	// 			Error:   "",
	// 		})
	// 	}
	// })

	if utils.FileIsExist("./config/flutter_studyclaw/") {
		router.StaticFS("/flutter_studyclaw", http.Dir("./config/flutter_studyclaw/"))
	}
	// 对权限的管理组
	auth := router.Group("/auth")
	// 用户登录的接口
	auth.POST("/login", userLogin())
	// 检查登录状态的token是否正确
	auth.POST("/check/:token", checkToken())

	// 对于用户可自定义挂载文件的目录
	if utils.FileIsExist("./config/dist/") {
		router.StaticFS("/dist", http.Dir("./config/dist/"))
	}

	config := router.Group("/config", check())

	config.GET("", configGet())
	config.POST("", configSet())
	config.GET("/file", configFileGet())
	config.POST("/file", configFileSet())

	// 对用户管理的组
	user := router.Group("/user", check())
	// 添加用户
	user.POST("", addUser())
	// 获取所以已登陆的用户
	user.GET("", getUsers())

	user.GET("/expired", getExpiredUser())

	// 删除用户
	user.DELETE("", deleteUser())

	// 获取用户成绩
	router.GET("/score", getScore())
	// 让一个用户开始学习
	router.POST("/study", study())
	// 让一个用户停止学习
	router.POST("/stop_study", check(), stopStudy())
	// 获取程序当天的运行日志
	router.GET("/log", check(), getLog())

	// 登录学习平台的三个接口
	router.GET("/sign/", sign())
	router.GET("/login/*proxyPath", generate())
	router.POST("/login/*proxyPath", check(), generate())
	return router
}

func check() gin.HandlerFunc {
	config := conf.GetConfig()
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		token = strings.Split(token, " ")[1]
		if token == "" {
			ctx.JSON(401, Resp{
				Code:    401,
				Message: "the auth fail",
				Data:    nil,
				Success: false,
				Error:   "",
			})
			ctx.Abort()
		} else if utils.StrMd5(config.Web.Account+config.Web.Password) == token {
			ctx.Set("level", 1)
			ctx.Set("token", token)
			ctx.Next()
		} else if checkCommonUser(token) {
			ctx.Set("level", 2)
			ctx.Set("token", token)
			ctx.Next()
		} else {
			ctx.JSON(401, Resp{
				Code:    401,
				Message: "the auth fail",
				Data:    nil,
				Success: false,
				Error:   "",
			})
			ctx.Abort()
		}
	}
}

func checkCommonUser(token string) bool {
	config := conf.GetConfig()
	for key, value := range config.Web.CommonUser {
		if token == utils.StrMd5(key+value) {
			return true
		}
	}
	return false
}
