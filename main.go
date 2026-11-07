package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	ctx = context.Background()
	rdb *redis.Client
	db  *gorm.DB
)

func initDependencies() error {
	// 1. Kết nối MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// 2. Kết nối Redis
	redisAddr := fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	return nil
}

func main() {
	if err := initDependencies(); err != nil {
		fmt.Printf("Init error: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()

	// Route kiểm tra sức khỏe hệ thống
	r.GET("/health", func(c *gin.Context) {
		redisStatus := "OK"
		if err := rdb.Ping(ctx).Err(); err != nil {
			redisStatus = "DOWN"
		}

		sqlDB, err := db.DB()
		mysqlStatus := "OK"
		if err != nil || sqlDB.Ping() != nil {
			mysqlStatus = "DOWN"
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "running",
			"mysql":  mysqlStatus,
			"redis":  redisStatus,
		})
	})

	// Ví dụ tăng lượt truy cập qua Redis
	r.GET("/visit", func(c *gin.Context) {
		visits, err := rdb.Incr(ctx, "page_views").Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"page_views": visits})
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
