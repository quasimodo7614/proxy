package user

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"proxy/jwt"
	"proxy/model"
	"time"
)

func User() gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Println("read body err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"read body err ": err.Error(),
			})
			return
		}
		req := &model.User{}
		// todo ： 验证用户密码
		err = json.Unmarshal(bodyBytes, req)
		if err != nil {
			log.Println("unmarshal err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"read body err ": err.Error(),
			})
			return
		}
		token, err := jwt.GenJwt(24 * time.Hour)
		if err != nil {
			log.Println("jwt gen err: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"token err ": err.Error(),
			})
			return
		}
		resp := &model.GenTokenResp{
			Jwt: token,
		}
		c.JSON(http.StatusOK, resp)
		return

	}

}
